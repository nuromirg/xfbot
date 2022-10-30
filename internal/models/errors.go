package models

import "errors"

var (
	ErrFailedToInitSlashEvent         = errors.New("failed to init slash event: session must register application commands")
	ErrFailedToRespondSlashEvent      = errors.New("failed to respond from slash event")
	ErrEnterTheVoiceChannelFirst      = errors.New("user isn't in the voice channel")
	ErrFailedToGetSongFromYoutube     = errors.New("unable to get song from youtube")
	ErrInternalError                  = errors.New("internal error")
	ErrFailedToGetSongMedia           = errors.New("unable to get song media")
	ErrFailedToGetPlaylistMedia       = errors.New("unable to get playlist media")
	ErrOptionDoesntContainYoutubeLink = errors.New("option doesn't contain youtube link")
)
