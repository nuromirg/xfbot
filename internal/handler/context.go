package handler

import (
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
	"xfbot/internal/bot"
	"xfbot/internal/pkg"
)

type Context struct {
	Session      *discordgo.Session
	Guild        *discordgo.Guild
	VoiceChannel *discordgo.Channel
	TextChannel  *discordgo.Channel
	User         *discordgo.User
	Message      *discordgo.MessageCreate
	Sessions     *bot.Sessions
	Handler      *Handler
	Youtube      *pkg.Youtube
	Args         []string
}

func NewContext(discordSession *discordgo.Session,
	discordGuild *discordgo.Guild,
	discordTextChannel *discordgo.Channel,
	discordUser *discordgo.User,
	discordMessage *discordgo.MessageCreate,
	sessions *bot.Sessions,
	handler *Handler,
	youtube *pkg.Youtube,
	args []string) *Context {

	return &Context{
		Session:     discordSession,
		Guild:       discordGuild,
		TextChannel: discordTextChannel,
		User:        discordUser,
		Message:     discordMessage,
		Sessions:    sessions,
		Handler:     handler,
		Youtube:     youtube,
		Args:        args,
	}
}

func (c *Context) Reply(resp string) (*discordgo.Message, error) {
	msg, err := c.Session.ChannelMessageSend(c.TextChannel.ID, resp)
	if err != nil {
		logrus.Errorf("could not send message. Error: %s", err.Error())
		return nil, err
	}
	return msg, nil
}

func (c *Context) GetVoiceChannel() *discordgo.Channel {
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
