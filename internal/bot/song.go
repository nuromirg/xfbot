package bot

import (
	"os/exec"
	"strconv"
	"xfbot/internal/core"
)

const signedLittleEndianInt16 = "s16le"

type Song struct {
	Media string
	Title string
	ID    string
}

func NewSong(media, title, id string) Song {
	return Song{
		Media: media,
		Title: title,
		ID:    id,
	}
}

func (s *Song) Pipe() *exec.Cmd {
	return exec.Command("ffmpeg", "-i", s.Media, "-f", signedLittleEndianInt16, "-ar", strconv.Itoa(core.FrameRate), "-ac",
		strconv.Itoa(core.Channels), "pipe:1")
}
