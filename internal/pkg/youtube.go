package pkg

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
)

const (
	ErrorType = iota - 1
	VideoType
	PlaylistType
)

type (
	videoResponse struct {
		Formats []struct {
			Url string `json:"url"`
		} `json:"formats"`
		Title string `json:"title"`
	}

	VideoResult struct {
		Media string
		Title string
	}

	PlaylistVideo struct {
		ID string `json:"id"`
	}

	YTSearchContent struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		ChannelTitle string `json:"channel_title"`
		Duration     string `json:"duration"`
	}

	ytApiResponse struct {
		IsError bool              `json:"error"`
		Content []YTSearchContent `json:"content"`
	}

	Youtube struct {
		BaseURL string
	}
)

func (yt Youtube) getType(input string) int {
	if strings.Contains(input, "upload_date") {
		return VideoType
	}

	if strings.Contains(input, "_type") {
		return PlaylistType
	}

	return ErrorType
}

func (yt Youtube) Get(input string) (int, *string, error) {
	command := exec.Command("youtube-dl", "--skip-download", "--print-json", "--flat-playlist", input)

	var out bytes.Buffer
	command.Stdout = &out
	if err := command.Run(); err != nil {
		return ErrorType, nil, err
	}

	str := out.String()
	return yt.getType(str), &str, nil
}

func (yt Youtube) Video(input string) (*VideoResult, error) {
	var resp videoResponse
	err := json.Unmarshal([]byte(input), &resp)
	if err != nil {
		return nil, err
	}
	return &VideoResult{resp.Formats[0].Url, resp.Title}, nil
}

func (yt Youtube) Playlist(input string) (*[]PlaylistVideo, error) {
	lines := strings.Split(input, "\n")
	videos := make([]PlaylistVideo, 0)

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var video PlaylistVideo
		if err := json.Unmarshal([]byte(line), &video); err != nil {
			return nil, err
		}
		videos = append(videos, video)
	}

	return &videos, nil
}

func (yt Youtube) buildUrl(query string) (*string, error) {
	base := yt.BaseURL + "/youtube/v2/search"

	address, err := url.Parse(base)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Add("search", query)
	address.RawQuery = params.Encode()
	str := address.String()

	return &str, nil
}

func (yt Youtube) Search(query string) ([]YTSearchContent, error) {
	addr, err := yt.buildUrl(query)
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(*addr)
	if err != nil {
		return nil, err
	}

	var apiResp ytApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	return apiResp.Content, nil
}
