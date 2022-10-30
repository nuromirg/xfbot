package bot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
	"xfbot/internal/core"
)

type Sessions map[string]*Session

type Session struct {
	Queue     *Queue
	GuildID   string
	ChannelID string
	conn      *core.Connection
}

func NewSession(conn *core.Connection, guildID, channelID string) *Session {
	return &Session{
		Queue:     NewQueue(),
		GuildID:   guildID,
		ChannelID: channelID,
		conn:      conn,
	}
}

func (s *Session) Play(song Song) {
	if err := s.conn.Play(song.Pipe()); err != nil {
		return
	}
}

func (s *Session) Stop() {
	s.conn.Stop()
}

func (s *Session) Pause() {
	s.conn.Pause()
}

func (s *Session) Resume() {
	s.conn.Resume()
}

func (ss Sessions) GetByGuild(guildID string) *Session {
	for _, s := range ss {
		if s.GuildID == guildID {
			return s
		}
	}

	return nil
}

func (ss Sessions) GetByChannel(channelId string) (*Session, bool) {
	s, found := ss[channelId]
	return s, found
}

func (ss Sessions) Join(discord *discordgo.Session, guildID, channelID string,
	isMuted bool, deafened bool) (*Session, error) {
	vc, err := discord.ChannelVoiceJoin(guildID, channelID, isMuted, deafened)
	if err != nil {
		return nil, err
	}

	s := NewSession(core.New(vc), guildID, channelID)
	ss[channelID] = s

	return s, nil
}

func (ss Sessions) Leave(session Session) {
	session.conn.Stop()

	if err := session.conn.Disconnect(); err != nil {
		logrus.Error("failed to disconnect from voice channel. Error: %s", err.Error())
		return
	}

	delete(ss, session.ChannelID)
}
