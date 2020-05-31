package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
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
var coolDown float64
var spamLimit int

// Create a temporary storage to check the last message time of an user, to prevent spam. This will also be a cooldown and will be pushed to
// the proper database when it's made,
//
var lastMessageList map[string]*lastMessage

type lastMessage struct {
	Timestamp   time.Time
	Message     *discordgo.MessageCreate
	LastWasSpam bool
	Count       int
}

func (lm *lastMessage) ToggleTrue() {
	lm.LastWasSpam = true
}

func (lm *lastMessage) ToggleFalse() {
	lm.LastWasSpam = false
}

func (lm *lastMessage) Reset() {
	lm.Count = 0
}

func (lm *lastMessage) Up() {
	lm.Count += 1
}

func (lm *lastMessage) SetTS(ts time.Time) {
	lm.Timestamp = ts
}

func (lm *lastMessage) SetMessage(message *discordgo.MessageCreate) {
	lm.Message = message
}

func (lm *lastMessage) CoolDown() bool {
	since := time.Since(lm.Timestamp).Seconds()
	if since < float64(coolDown) {
		return true
	}
	return false
}

func init() {
	// Perform initial operations, opening databases and checking config files
	// Greet in console when ready
	lastMessageList = make(map[string]*lastMessage)
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
	coolDownRaw, _ := strconv.Atoi(config.Config("CoolDownTime", "Default"))
	coolDown = float64(coolDownRaw)
	spamLimit, _ = strconv.Atoi(config.Config("SpamLimit", "Default"))
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
	// Check lastMessageList for spam

	if message.Author.Bot {
		return
	}

	messageTime := time.Now()

	if lm, ok := lastMessageList[message.Author.ID]; ok && lm.CoolDown() {
		if lm.LastWasSpam && lm.Count >= spamLimit {
			client.ChannelMessageSend(message.ChannelID, "Don't spam! "+message.Author.String())
			warnings, err := databases.SafeQuery(`select spam_warnings from user where id=?`, message.Author.ID)
			if err == nil {
				warning := warnings[0]
				intWarning, _ := strconv.Atoi(warning)
				maxWarningsRaw := config.Config("MaxWarnings", "Default")
				maxWarnings, _ := strconv.Atoi(maxWarningsRaw)
				if intWarning >= maxWarnings {
					client.ChannelMessageSend(message.ChannelID, "Muting user "+message.Author.String())
					currentGuild, _ := client.Guild(message.GuildID)
					muted := commands.Mute(message.Author, currentGuild)
					if muted {
						client.ChannelMessageSend(message.ChannelID, "Successfully muted user "+message.Author.String())
					}
				} else {
					databases.SafeExec(`update user set spam_warnings=spam_warnings+1 where id=?`, message.Author.ID)
				}
			}
			lm.Reset()
			lm.ToggleFalse()
		} else if lm.LastWasSpam && lm.Count < spamLimit && lm.Count > 0 {
			lm.Up()
		} else {
			lm.ToggleTrue()
			lm.Up()
		}
		lm.SetTS(messageTime)
		lm.SetMessage(message)
		return
	} else if ok {
		lm.ToggleFalse()
		lm.Reset()
		lm.SetTS(messageTime)
		lm.SetMessage(message)
	} else if !ok {
		lastMessageList[message.Author.ID] = new(lastMessage)
		rows, err := databases.SafeQuery(`select * from user where id=?`, message.Author.ID)
		if err != nil {
			return
		}
		if len(rows) == 0 {
			_, err := databases.SafeExec(`insert into user values(?,?,?)`, message.Author.ID, 0, "Default")
			if err != nil {
				return
			}
		}

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
