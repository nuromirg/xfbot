package handler

import (
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	"strconv"
	"xfbot/internal/bot"
	"xfbot/internal/pkg"
)

const (
	songsPerPage = 10
)

type Handler struct {
	commands map[string]Command
}

type HandleFunc func(Context)

type Command struct {
	handleFunc HandleFunc
	help       string
}

func NewHandler() *Handler {
	return &Handler{
		commands: make(map[string]Command),
	}
}

func (h *Handler) Init() {
	h.registerCommand("add", Add, "\"Add a song to the queue `xf add <youtube-link>`\"")
	h.registerCommand("join", Join, "\"Join a channel using `xf join`\"")
	h.registerCommand("play", Play, "\"Play songs from the queue `xf play <youtube-link>`\"")
	h.registerCommand("skip", Skip, "\"Skip songs in the queue `xf skip`\"")
	h.registerCommand("stop", Stop, "\"Stop current song and the queue `xf stop`\"")
	h.registerCommand("queue", Queue, "\"List current song queue `xf queue`\"")
}

func (h *Handler) registerCommand(arg string, handle HandleFunc, help string) {
	command := Command{
		handleFunc: handle,
		help:       help,
	}
	h.commands[arg] = command
}

func (h *Handler) List() map[string]Command {
	return h.commands
}

func (h *Handler) Get(arg string) (HandleFunc, bool) {
	command, found := h.commands[arg]
	return command.handleFunc, found
}

func Add(ctx Context) {
	if len(ctx.Args) == 0 {
		_, err := ctx.Reply("Usage: `xf add <song_url>`\nSupporting only Youtube (Spotify will be later).")
		if err != nil {
			return
		}
		return
	}

	session := ctx.Sessions.GetByGuild(ctx.Guild.ID)
	if session == nil {
		ctx.Reply("Enter the voice channel first. Or use `xf join` instead.")
		return
	}

	msg, _ := ctx.Reply("Adding songs to queue...")

	for _, arg := range ctx.Args {
		got, input, err := ctx.Youtube.Get(arg)
		if err != nil {
			ctx.Reply("unable to get song from youtube")
			logrus.Errorf("error when trying to get input. Error: %s", err.Error())
			return
		}
		switch got {
		case pkg.ErrorType:
			ctx.Reply("unable to get song from youtube")
			if err != nil {
				return
			}
			logrus.Errorf("Got ERROR_TYPE: %v", got)
			return
		case pkg.VideoType:
			{
				video, err := ctx.Youtube.Video(*input)
				if err != nil {
					ctx.Reply("An error occured!")
					fmt.Println("error getting video1,", err)
					return
				}
				song := bot.NewSong(video.Media, video.Title, arg)
				session.Queue.Add(song)
				ctx.Session.ChannelMessageEdit(ctx.TextChannel.ID, msg.ID, "Added `"+song.Title+"` to the song queue."+
					" Use `xf play` to start playing the songs! To see the song queue, use `xf queue`.")
			}
		case pkg.PlaylistType:
			playlist, err := ctx.Youtube.Playlist(*input)
			if err != nil {
				ctx.Reply("unable to get song from youtube")
				logrus.Errorf("error when trying to get playlist. Error: %s", err.Error())
				return
			}
			for _, video := range *playlist {
				id := video.ID
				_, i, err := ctx.Youtube.Get(id)
				if err != nil {
					ctx.Reply("unable to get song from youtube")
					logrus.Errorf("error when trying to get video from playlist. Error: %s", err.Error())
					continue
				}

				youtubeVideo, err := ctx.Youtube.Video(*i)
				if err != nil {
					ctx.Reply("unable to get song from youtube")
					logrus.Errorf("error when trying to get video from playlist. Error: %s", err.Error())
					return
				}
				song := bot.NewSong(youtubeVideo.Media, youtubeVideo.Title, arg)
				session.Queue.Add(song)
			}
			ctx.Reply("Added songs to your playlist. Use `xf play` to start playing the songs. " +
				"To see the song queue, use `xf queue`.")
			return
		}
	}
}

func Join(ctx Context) {
	if ctx.Sessions.GetByGuild(ctx.Guild.ID) != nil {
		ctx.Reply("Already in the channel! Use `xf leave` for the bot to disconnect.")
		return
	}
	vc := ctx.GetVoiceChannel()
	if vc == nil {
		ctx.Reply("You must be in a voice channel to use the bot.")
		return
	}

	_, err := ctx.Sessions.Join(ctx.Session, ctx.Guild.ID, vc.ID, false, true)
	if err != nil {
		ctx.Reply("An error occurred :(")
		return
	}
}

func Play(ctx Context) {
	var err error
	s := ctx.Sessions.GetByGuild(ctx.Guild.ID)
	if s == nil {
		vc := ctx.GetVoiceChannel()
		if vc == nil {
			ctx.Reply("You must be in a voice channel to use the bot.")
			return
		}

		s, err = ctx.Sessions.Join(ctx.Session, ctx.Guild.ID, vc.ID, false, true)
		if err != nil {
			ctx.Reply("An error occurred :(")
			return
		}
	}

	queue := s.Queue

	if !queue.HasNext() && len(ctx.Args) == 0 {
		ctx.Reply("Queue is empty! Use `xf play <youtube-link>` instead.")
		return
	}

	if len(ctx.Args) != 0 {
		Add(ctx)
		if queue.Current() != nil {
			return
		}
	}

	go queue.Play(s)
}

func Skip(ctx Context) {
	s := ctx.Sessions.GetByGuild(ctx.Guild.ID)
	if s == nil {
		ctx.Reply("I'm not in the voice channel. To make the bot join one, use `xf play <youtube-link>`.")
		return
	}
	s.Stop()
	ctx.Reply("Song skipped")
}

func Stop(ctx Context) {
	s := ctx.Sessions.GetByGuild(ctx.Guild.ID)
	if s == nil {
		ctx.Reply("I'm not in the voice channel. To make the bot join one, use `xf play <youtube-link>`.")
		return
	}
	if s.Queue.HasNext() {
		s.Queue.Clear()
	}
	s.Stop()
}

func Queue(ctx Context) {
	s := ctx.Sessions.GetByGuild(ctx.Guild.ID)
	if s == nil {
		ctx.Reply("I'm not in the voice channel. To make the bot join one, use `xf play <youtube-link>`.")
		return
	}

	sessionQueue := s.Queue
	queue := sessionQueue.Get()
	queueLength := len(queue)

	if queueLength == 0 && sessionQueue.Current() == nil {
		ctx.Reply("Queue is empty. Add a song with `xf play <youtube-link>`.")
		return
	}

	var buf bytes.Buffer
	if sessionQueue.Current() != nil {
		buf.WriteString(fmt.Sprintf("> **Current song:** `%s`\n", sessionQueue.Current().Title))
	}

	pages := queueLength / songsPerPage
	if len(ctx.Args) == 0 {
		var resp string
		if queueLength > songsPerPage {
			resp = paging(queue[:songsPerPage], buf, 1, 1, 0)
		} else {
			resp = paging(queue[:], buf, 1, pages, 0)
		}
		ctx.Reply(resp)
		return
	}

	page, err := strconv.Atoi(ctx.Args[0])
	if err != nil {
		ctx.Reply("Invalid page `" + ctx.Args[0] + "`. Usage: `xf queue <page>`")
		return
	}

	if page < 1 || page > (pages+1) {
		ctx.Reply(fmt.Sprintf("Invalid page `%d`. Choose between 1 and %d.", page, pages+1))
		return
	}

	var lowerBound int
	if page == 1 {
		lowerBound = 0
	} else {
		lowerBound = (page - 1) * songsPerPage
	}

	upperBound := page * songsPerPage
	if upperBound > queueLength {
		upperBound = queueLength
	}

	ctx.Reply(paging(queue[lowerBound:upperBound], buf, page+1, pages+1, lowerBound))
}

func paging(queue []bot.Song, buf bytes.Buffer, page, pages, start int) string {
	for index, song := range queue {
		buf.WriteString(fmt.Sprintf("\n`%d. %s`", start+index+1, song.Title))
	}

	switch {
	case pages > page:
		buf.WriteString(fmt.Sprintf("\n\nPage **%d** of **%d**. To view the next page, use `xf queue %d`.", page, pages, page+1))
	case pages == page:
		buf.WriteString(fmt.Sprintf("\n\nPage **%d** of **%d**.", page, pages))
	case pages < page:
		buf.WriteString(fmt.Sprintf("\n\nPage **%d** of **%d**.", 1, 1))
	}

	return buf.String()
}
