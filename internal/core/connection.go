package core

import (
	"bufio"
	"encoding/binary"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/hraban/opus.v2"
	"io"
	"os/exec"
	"sync"
)

const (
	Channels   int = 2
	FrameRate  int = 48000
	frameSize  int = 960
	readerSize int = 16384
	bufferSize     = frameSize * 4
)

type Connection struct {
	voiceConnection *discordgo.VoiceConnection
	send            chan []int16
	lock            sync.Mutex
	sentPCM         bool
	stopRunning     bool
	isPlaying       bool
}

func New(vc *discordgo.VoiceConnection) *Connection {
	conn := new(Connection)
	conn.voiceConnection = vc
	return conn
}

func (c *Connection) Disconnect() error {
	if err := c.voiceConnection.Disconnect(); err != nil {
		return err
	}
	return nil
}

func (c *Connection) sendPCM(voice *discordgo.VoiceConnection, pcm <-chan []int16) {
	c.lock.Lock()
	if c.sentPCM || pcm == nil {
		c.lock.Unlock()
		return
	}

	c.sentPCM = true
	c.lock.Unlock()
	defer func() {
		c.sentPCM = false
	}()

	encoder, err := opus.NewEncoder(FrameRate, Channels, opus.AppAudio)
	if err != nil {
		logrus.Errorf("failed to make new encoder. IsError: %s", err.Error())
		return
	}

	for {
		receive, ok := <-pcm
		if !ok {
			logrus.Info("PCM channel closed")
			return
		}

		data := make([]byte, bufferSize)
		n, err := encoder.Encode(receive, data)
		if err != nil {
			logrus.Error("Encoding error,", err)
			return
		}

		if !voice.Ready || voice.OpusSend == nil {
			logrus.Errorf("Bot is not ready for opus packets. %+v : %+v", voice.Ready, voice.OpusSend)
			return
		}

		voice.OpusSend <- data[:n]
	}
}

func (c *Connection) Play(ffmpeg *exec.Cmd) error {
	if c.isPlaying {
		return errors.New("song is playing right now")
	}
	c.stopRunning = false

	ffmpegOut, err := ffmpeg.StdoutPipe()
	if err != nil {
		return err
	}

	buf := bufio.NewReaderSize(ffmpegOut, readerSize)
	if err := ffmpeg.Start(); err != nil {
		return err
	}

	c.isPlaying = true
	defer func() {
		c.isPlaying = false
	}()

	c.voiceConnection.Speaking(true)
	defer c.voiceConnection.Speaking(false)
	if c.send == nil {
		c.send = make(chan []int16, 2)
	}

	go c.sendPCM(c.voiceConnection, c.send)
	for {
		if c.stopRunning {
			ffmpeg.Process.Kill()
			break
		}

		audioBuffer := make([]int16, frameSize*Channels)
		if err := binary.Read(buf, binary.LittleEndian, &audioBuffer); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}

			return err
		}

		c.send <- audioBuffer
	}

	return nil
}

func (c *Connection) Pause() {
	c.isPlaying = false
	c.voiceConnection.Speaking(false)
}

func (c *Connection) Resume() {
	c.isPlaying = true
	c.voiceConnection.Speaking(true)
}

func (c *Connection) Stop() {
	c.stopRunning = true
	c.isPlaying = false
}
