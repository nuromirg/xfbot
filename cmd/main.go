package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"log"
	"os"
	"strings"
	"xfbot/internal/bot"
	"xfbot/internal/handler"
	"xfbot/internal/models"
	"xfbot/internal/pkg"
)

func main() {
	if err := godotenv.Load("./cmd/test.env"); err != nil {
		log.Fatalf("error occurred when reading env. Error: %s", err.Error())
	}

	token := os.Getenv("BOT_TOKEN")
	prefix := os.Getenv("PREFIX")
	youtubeURL := os.Getenv("YOUTUBE_URL")
	appID := os.Getenv("APP_ID")

	h := handler.NewHandler()
	h.Init()

	sessions := make(bot.Sessions, 0)
	ytClient := &pkg.Youtube{
		BaseURL: youtubeURL,
	}

	discord, err := discordgo.New("Bot " + token)
	if err != nil {
		logrus.Fatalf("couldn't create discord session. Error: %s\n", err.Error())
	}

	user, err := discord.User("@me")
	if err != nil {
		logrus.Fatalf("can't get user details. Error: %s\n", err.Error())
	}

	botID := user.ID

	registeredCommands := make([]*discordgo.ApplicationCommand, 0)
	for _, cmd := range models.Commands {
		registered, err := discord.ApplicationCommandCreate(appID, "", cmd)
		if err != nil {
			log.Panicf("cannot create '%v' command: %v", cmd.Name, err)
		}

		registeredCommands = append(registeredCommands, registered)
	}

	slashEvents := handler.NewSlashEvents(registeredCommands)
	if err := slashEvents.Init(); err != nil {
		return
	}

	events := slashEvents.Events()
	discord.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		var author *discordgo.User
		if i.User != nil {
			author = i.User
			if author.ID == botID || i.User.Bot {
				logrus.Warnf("bot cannot execute slash commands. Username of the bot is %s\n", i.User.Username)
				return
			}

		} else {
			author = i.Member.User
			if author.ID == botID || i.Member.User.Bot {
				logrus.Warnf("bot cannot execute slash commands. Username of the bot is %s\n", i.Member.User.Username)
				return
			}
		}

		channel, err := s.State.Channel(i.ChannelID)
		if err != nil {
			logrus.Errorf("cannot get the channel: %s\n", err.Error())
			return
		}

		guild, err := s.State.Guild(channel.GuildID)
		if err != nil {
			logrus.Errorf("cannot get the guild.: %s\n", err.Error())
			return
		}

		eCtx := handler.EventContext{
			Session:     s,
			Guild:       guild,
			Interaction: i.Interaction,
			User:        author,
			Sessions:    &sessions,
			Youtube:     ytClient,
		}

		if h, ok := events[i.ApplicationCommandData().Name]; ok {
			h(s, i, eCtx)
		}
	})

	discord.AddHandler(func(session *discordgo.Session, message *discordgo.MessageCreate) {
		author := message.Author
		if author.ID == botID || author.Bot {
			return
		}

		content := message.Content
		if len(content) <= len(prefix) {
			logrus.Errorf("empy message content")
			return
		}

		if content[:len(prefix)] != prefix {
			return
		}

		content = content[len(prefix):]
		if len(content) < 1 {
			return
		}

		args := strings.Fields(content)
		name := strings.ToLower(args[0])

		command, isFound := h.Get(name)
		if !isFound {
			return
		}

		channel, err := session.State.Channel(message.ChannelID)
		if err != nil {
			logrus.Errorf("cannot get channel. Error: %s\n", err.Error())
			return
		}

		guild, err := session.State.Guild(channel.GuildID)
		if err != nil {
			logrus.Errorf("cannot get guild. Error: %s\n", err.Error())
			return
		}

		ctx := handler.NewContext(session, guild, channel, author, message, &sessions, h, ytClient, args)
		ctx.Args = args[1:]
		exec := command
		exec(*ctx)
	})

	discord.AddHandler(func(discord *discordgo.Session, ready *discordgo.Ready) {
		if err := discord.UpdateGameStatus(0, fmt.Sprintf("%shelp", prefix)); err != nil {
			logrus.Errorf("can't update to default status. Error: %s\n", err.Error())
		}
		guilds := discord.State.Guilds
		logrus.Print("In ", len(guilds), " servers.")
	})

	defer func(session *discordgo.Session) {
		err := session.Close()
		if err != nil {
			logrus.Errorf("can't close a websocket. Error: %s\n", err.Error())
		}
	}(discord)

	if err := discord.Open(); err != nil {
		logrus.Fatal("couldn't open connection. Error: %s\n", err.Error())
	}

	logrus.Println("Started")
	<-make(chan struct{})
}
