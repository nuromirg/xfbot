package models

import "github.com/bwmarrin/discordgo"

var (
	PingEventName = "ping"
	PlayEventName = "play"
)

var (
	Commands = []*discordgo.ApplicationCommand{
		{
			Name:        PingEventName,
			Description: "request \"ping\" from server",
		},
		{
			Name:        PlayEventName,
			Description: "command that plays song from link",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "link",
					Description: "link to the resource to be played",
					Required:    true,
				},
			},
		},
	}
)
