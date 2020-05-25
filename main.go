package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	config "uneex/config"
	commands "uneex/core"
	databases "uneex/databases"

	"github.com/bwmarrin/discordgo"
	"github.com/go-ini/ini"
	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/gographics/imagick.v3/imagick"
)

// Globals

var sc chan os.Signal = make(chan os.Signal, 1)

func init() {
	// Perform initial operations, opening databases and checking config files
	// Greet in console when ready
	fmt.Println("Starting Uneex bot")
	imagick.Initialize()
}

func main() {
	// Open config ini file by marshaling with ini.Load()
	defer imagick.Terminate()
	var err error
	config.Conf, _ = ini.Load("config/config.ini")
	fmt.Println(config.Config("Version"))
	// Open database for use
	databases.Database, err = sql.Open("mysql", fmt.Sprintf("%s:%s@/%s", config.Config("User", "Maria"), config.Config("Password", "Maria"), config.Config("Name", "Maria")))
	if err != nil {
		panic(err)
	}
	// Create discord bot session and handle any errors adequately
	client, err := discordgo.New("Bot " + config.Config("Token", "Owner"))
	if err != nil {
		panic(err)
	}
	// Handlers
	client.AddHandler(OnMessageCreate)
	// Client session should be open now if no errors had occurred
	err = client.Open()
	// Start cron handler
	// stop := make(chan bool)
	// FIXME Temporarily disable cron worker for debug
	// go cron.Worker(stop, client)
	if err != nil {
		panic(err)
	}
	guilds := client.State.Guilds
	game := &discordgo.Game{Name: strconv.Itoa(len(guilds)) + " Guilds!"}
	status := &discordgo.UpdateStatusData{Game: game}
	client.UpdateStatusComplex(*status)
	fmt.Println("Client started")
	// Handle syscalls to quit bot gracefully and close database connection
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Close connections
	databases.Database.Close()
	client.Close()
}

// Handlers
//
func OnMessageCreate(client *discordgo.Session, message *discordgo.MessageCreate) {
	// Check if the message contains the bot's prefix
	if message.Author.Bot {
		return
	}
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
	// Handle the rest of the command, it should have the prefix trimmed as well as the mentions
	commands.CommandHandler(client, message, Content, Mentions, sc)
}

func PrefixHandler(message *discordgo.MessageCreate) (bool, string) {
	if strings.HasPrefix(message.Content, config.Config("Prefix", "Default")) {
		// Strip prefix
		withoutPrefix := strings.TrimPrefix(message.ContentWithMentionsReplaced(), config.Config("Prefix", "Default"))
		return true, withoutPrefix
	}
	return false, ""
}
