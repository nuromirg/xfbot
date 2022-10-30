package handler

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
	"strings"
	"xfbot/internal/bot"
	"xfbot/internal/models"
	"xfbot/internal/pkg"
)

const youtubeURL = "https://youtube.com/"

type (
	InteractionFunc    func(s *discordgo.Session, i *discordgo.InteractionCreate, eCtx EventContext)
	SlashCommandEvents map[string]InteractionFunc
)

type EventContext struct {
	Session      *discordgo.Session
	Guild        *discordgo.Guild
	VoiceChannel *discordgo.Channel
	Interaction  *discordgo.Interaction
	User         *discordgo.User
	Sessions     *bot.Sessions
	Youtube      *pkg.Youtube
}

func (c *EventContext) GetVoiceChannel() *discordgo.Channel {
	if c.VoiceChannel != nil {
		return c.VoiceChannel
	}

	for _, state := range c.Guild.VoiceStates {
		if state.UserID == c.User.ID {
			channel, _ := c.Session.State.Channel(state.ChannelID)
			c.VoiceChannel = channel
			return channel
		}
	}

	return nil
}

type Slash interface {
	Init() error
	Events() SlashCommandEvents
}

type slashEvents struct {
	registeredCommands []*discordgo.ApplicationCommand
	eventsRegistry     map[string]InteractionFunc
}

func NewSlashEvents(registeredCommands []*discordgo.ApplicationCommand) Slash {
	return &slashEvents{
		registeredCommands: registeredCommands,
	}
}

func (h *slashEvents) Init() error {
	h.eventsRegistry = SlashCommandEvents{
		models.PingEventName: func(s *discordgo.Session, i *discordgo.InteractionCreate, eCtx EventContext) {
			if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "pong",
				},
			}); err != nil {
				logrus.Fatal(models.ErrFailedToInitSlashEvent, err)
			}
		},

		models.PlayEventName: func(s *discordgo.Session, i *discordgo.InteractionCreate, eCtx EventContext) {
			var err error
			botSession := eCtx.Sessions.GetByGuild(eCtx.Guild.ID)
			if botSession == nil {
				vc := eCtx.GetVoiceChannel()
				if vc == nil {
					if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "You need to enter the voice channel first 😔",
						},
					}); err != nil {
						logrus.Fatal(models.ErrFailedToRespondSlashEvent, err)
					}
					return
				}

				botSession, err = eCtx.Sessions.Join(eCtx.Session, eCtx.Guild.ID, vc.ID, false, true)
				if err != nil {
					if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Internal error 😔",
						},
					}); err != nil {
						logrus.Fatal(models.ErrFailedToRespondSlashEvent, err)
					}
					return
				}
			}

			queue := botSession.Queue
			opt := i.ApplicationCommandData().Options[0]

			if opt != nil {
				if err := addSongToQueue(eCtx, opt); err != nil {

					return
				}

				if queue.Current() != nil {
					return
				}
			}

			if !queue.HasNext() || opt == nil {
				if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Queue or slash command option is empty. Try to use `<youtube-link>` as an option instead 😊",
					},
				}); err != nil {
					logrus.Fatal(models.ErrFailedToRespondSlashEvent, err)
				}
				return
			}

			go queue.Play(botSession)
		},
	}

	return nil
}

func (h *slashEvents) Events() SlashCommandEvents {
	return h.eventsRegistry
}

func addSongToQueue(eCtx EventContext, opt *discordgo.ApplicationCommandInteractionDataOption) error {
	if opt == nil {
		if !strings.Contains(opt.StringValue(), youtubeURL) {
			return models.ErrOptionDoesntContainYoutubeLink
		}
	}

	botSession := eCtx.Sessions.GetByGuild(eCtx.Guild.ID)
	if botSession == nil {
		return models.ErrEnterTheVoiceChannelFirst
	}

	got, input, err := eCtx.Youtube.Get(opt.StringValue())
	if err != nil {
		logrus.Errorf("unable to get song from youtube: %s", err.Error())
		return models.ErrFailedToGetSongFromYoutube
	}

	msg, err := eCtx.Session.FollowupMessageCreate(eCtx.Interaction, true, &discordgo.WebhookParams{
		Content: "Adding songs to queue... 🎶",
	})

	switch got {
	case pkg.ErrorType:
		logrus.Errorf("unable to get song from youtube: %s", err.Error())
		return models.ErrFailedToGetSongFromYoutube
	case pkg.VideoType:
		{
			if input != nil {
				video, err := eCtx.Youtube.Video(*input)
				if err != nil {
					logrus.Errorf("unable to get song media: %s", err.Error())
					return models.ErrFailedToGetSongMedia
				}
				song := bot.NewSong(video.Media, video.Title, opt.StringValue())
				botSession.Queue.Add(song)

				msg, err = eCtx.Session.FollowupMessageEdit(eCtx.Interaction, msg.ID, &discordgo.WebhookEdit{
					Content: fmt.Sprintf("Added `\"+%s+\"` to the song queue 🎶", song.Title),
				})

			}

			return models.ErrInternalError
		}
	case pkg.PlaylistType:
		if input != nil {
			playlist, err := eCtx.Youtube.Playlist(*input)
			if err != nil {
				logrus.Errorf("unable to get playlist media: %s", err.Error())
				return models.ErrFailedToGetSongMedia
			}

			if playlist != nil {
				for _, video := range *playlist {
					id := video.ID
					_, i, err := eCtx.Youtube.Get(id)
					if err != nil {
						logrus.Errorf("error when trying to get video from playlist. Error: %s", err.Error())
						continue
					}

					youtubeVideo, err := eCtx.Youtube.Video(*i)
					if err != nil {
						logrus.Errorf("unable to get song media: %s", err.Error())
						return models.ErrFailedToGetSongMedia
					}
					song := bot.NewSong(youtubeVideo.Media, youtubeVideo.Title, opt.StringValue())
					botSession.Queue.Add(song)
				}
				msg, err = eCtx.Session.FollowupMessageEdit(eCtx.Interaction, msg.ID, &discordgo.WebhookEdit{
					Content: "Added songs to your playlist 🎶",
				})

				return nil
			}

		}

		return models.ErrInternalError
	}

	return nil
}
