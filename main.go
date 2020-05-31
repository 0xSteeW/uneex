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
	"github.com/jonas747/dshardmanager"
	"gopkg.in/gographics/imagick.v3/imagick"
)

// Globals

var sc chan os.Signal = make(chan os.Signal, 1)
var Manager *dshardmanager.Manager

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
	// client, err := discordgo.New("Bot " + config.Config("Token", "Owner"))
	Manager = dshardmanager.New("Bot " + config.Config("Token", "Owner"))
	if err != nil {
		panic(err)
	}
	// Handlers
	Manager.AddHandler(OnMessageCreate)
	Manager.Init()
	// Client session should be open now if no errors had occurred
	err = Manager.Start()
	// Start cron handler
	// stop := make(chan bool)
	// FIXME Temporarily disable cron worker for debug
	// go cron.Worker(stop, client)
	if err != nil {
		panic(err)
	}
	fmt.Println("Client started")
	ChangeStatus()
	// Handle syscalls to quit bot gracefully and close database connection
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Close connections
	databases.Database.Close()
	Manager.StopAll()
}

func ChangeStatus() {
	for _, session := range Manager.Sessions {
		totalGuilds := Manager.GetFullStatus().NumGuilds
		session.UpdateListeningStatus(strconv.Itoa(totalGuilds))
	}
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
	commands.CommandHandler(client, message, Content, Mentions, sc, Manager)
}

func PrefixHandler(message *discordgo.MessageCreate) (bool, string) {
	if strings.HasPrefix(message.Content, config.Config("Prefix", "Default")) {
		// Strip prefix
		withoutPrefix := strings.TrimPrefix(message.Content, config.Config("Prefix", "Default"))
		return true, withoutPrefix
	}
	return false, ""
}
