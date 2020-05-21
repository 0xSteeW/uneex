package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	commands "uneex/core"
	databases "uneex/databases"

	"github.com/bwmarrin/discordgo"
	"github.com/go-ini/ini"
)

// Globals

var conf *ini.File
var mutex = &sync.Mutex{}
var sc chan os.Signal = make(chan os.Signal, 1)

func Config(name string, section ...string) string {
	mutex.Lock()
	defer mutex.Unlock()
	if len(section) == 0 {
		return conf.Section("").Key(name).String()
	}
	return conf.Section(section[0]).Key(name).String()
}

func init() {
	// Perform initial operations, opening databases and checking config files
	fmt.Println(databases.Test())
	// Greet in console when ready
	fmt.Println("Starting Uneex bot")
}

func main() {
	// Open config ini file by marshaling with ini.Load()
	conf, _ = ini.Load("config.ini")
	fmt.Println(Config("Version"))
	client, err := discordgo.New("Bot " + Config("Token", "Owner"))
	if err != nil {
		panic(err)
	}
	// Handlers
	client.AddHandler(OnMessageCreate)
	// Client session should be open now if no errors had occurred
	err = client.Open()
	if err != nil {
		panic(err)
	}

	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	client.Close()
}

// Handlers
//
func OnMessageCreate(client *discordgo.Session, message *discordgo.MessageCreate) {
	// Check if the message contains the bot's prefix
	isPrefixed, trimmed := PrefixHandler(message)
	if !isPrefixed {
		return
	}
	// Continue
	// Useful variables inside the function
	Content := trimmed
	var Mentions []*discordgo.User
	if message.Mentions != nil {
		Mentions = message.Mentions
	}
	Mentions = Mentions // FIXME temporary assignment
	// Handle the rest of the command, it should have the prefix trimmed as well as the mentions
	CommandHandler(client, message, Content, Mentions)
}

func PrefixHandler(message *discordgo.MessageCreate) (bool, string) {
	if strings.HasPrefix(message.Content, Config("Prefix", "Default")) {
		// Strip prefix
		withoutPrefix := strings.TrimPrefix(message.ContentWithMentionsReplaced(), Config("Prefix", "Default"))
		return true, withoutPrefix
	}
	return false, ""
}

func CommandHandler(client *discordgo.Session, message *discordgo.MessageCreate, content string, mentions []*discordgo.User) {
	// Receive content with mentions stripped
	// Global variables to use
	origin := message.ChannelID
	exported := commands.ExportedSession{Client: client, Message: message}
	switch strings.ToLower(content) {
	case "ping":
		exported.Ping()
	case "shutdown":
		if Config("ID", "Owner") == message.Author.ID {
			exported.Shutdown()
			sc <- syscall.SIGTERM
		} else {
			client.ChannelMessageSend(origin, "Sorry, I don't think you have enough permissions to use this.")
		}
	default:
		return
	}
}
