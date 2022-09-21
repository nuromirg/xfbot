package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"log"
	"os"
	"strings"
	"xfbot/internal/bot"
	"xfbot/internal/handler"
	"xfbot/internal/pkg"
)

func main() {
	if err := godotenv.Load("./cmd/test.env"); err != nil {
		log.Fatalf("error occured when reading env. Error: %s", err.Error())
	}

	token := os.Getenv("BOT_TOKEN")
	// ownerID := os.Getenv("OWNER_ID")
	prefix := os.Getenv("PREFIX")
	youtubeURL := os.Getenv("YOUTUBE_URL")

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

	discord.AddHandler(func(session *discordgo.Session, message *discordgo.MessageCreate) {
		author := message.Author
		if author.ID == botID || author.Bot {
			return
		}

		content := message.Content
		if len(content) <= len(prefix) {
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
		if err := discord.UpdateGameStatus(0, "xf help"); err != nil {
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
