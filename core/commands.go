package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
	config "uneex/config"
	databases "uneex/databases"

	"github.com/bwmarrin/discordgo"
)

// Global core commands
//
//
var Client *discordgo.Session
var Message *discordgo.MessageCreate
var Content string
var Mentions []*discordgo.User
var SC *chan os.Signal

func Ping(buffer *Buffer) {
	latency := Client.HeartbeatLatency()
	var fields []*discordgo.MessageEmbedField
	fieldPingNormal := &discordgo.MessageEmbedField{Name: "Truncated", Value: latency.String()}
	fieldPingNanoseconds := &discordgo.MessageEmbedField{Name: "Nanoseconds", Value: strconv.FormatInt(latency.Nanoseconds(), 10)}
	fieldPingMicroseconds := &discordgo.MessageEmbedField{Name: "Microseconds", Value: strconv.FormatInt(latency.Microseconds(), 10)}
	fields = []*discordgo.MessageEmbedField{fieldPingNormal, fieldPingNanoseconds, fieldPingMicroseconds}
	embed := &discordgo.MessageEmbed{Title: "Pong!", Description: "Successful ping! Showing RTT:", Fields: fields}

	Client.ChannelMessageSendEmbed(Message.ChannelID, embed)
}

func Shutdown(buffer *Buffer) {
	fmt.Println("Shutting down...")
	Client.ChannelMessageSend(Message.ChannelID, "Shutdown requested, proceeding...")
}

func Cat(buffer *Buffer) {
	fmt.Println("Received cat")
	Client.ChannelMessageSend(Message.ChannelID, buffer.Content)
}

func Substitute(buffer *Buffer) {
	buffer.Content = strings.ReplaceAll(buffer.Content, "a", "7")
}

func NewCron(content string, buffer *Buffer) {
	timeStamp := strings.Split(content, " -c")[0]
	Client.ChannelMessageSend(Message.ChannelID, "`"+timeStamp+"`")
	timeStampParse, err := time.Parse(`Mon Jan 2 15:04:05 -0700 MST 2006`, timeStamp)
	if err != nil {
		Client.ChannelMessageSend(Message.ChannelID, "Sorry, the time format you provided isn't valid.")
		Client.ChannelMessageSend(Message.ChannelID, err.Error())
		return
	}
	remind := strings.Split(content, "-c")[1:]
	var remindLiteral string
	for _, word := range remind {
		remindLiteral = remindLiteral + word
	}
	userAlreadyExists, err := databases.SafeQuery(`select * from user where id=?`, Message.Author.ID)
	if err != nil {
		return
	}
	if len(userAlreadyExists) == 0 {
		databases.SafeExec(`insert into user values(?)`, Message.Author.ID)
	}
	databases.SafeExec(`insert into jobs values(?,?,?)`, timeStampParse, Message.Author.ID, remindLiteral)
	Client.ChannelMessageSend(Message.ChannelID, "Succesfully added remind for "+timeStampParse.String()+" with content: "+remindLiteral)
}

func RemoveCommand(cmd string) string {
	cmds := strings.Split(cmd, " ")
	return strings.TrimPrefix(cmd, cmds[0])
}

type Bufferable interface {
	Signal()
	Transfer(*Buffer)
}

type Buffer struct {
	Content string
	Pipes   []string
	Next    []string
}

func (buff *Buffer) Pop() {
	buff.Next = buff.Next[1:]
}

func Transfer(origin, destination *Buffer) {
	destination = origin
}

func (buff *Buffer) CreateWithPipes(content string) {
	buff.Pipes = strings.Split(content, "|")
	buff.Next = buff.Pipes
	buff.Clean()
}

func (buff *Buffer) Print() {
	var tmp string
	for _, p := range buff.Next {
		tmp = tmp + p
	}
	fmt.Println(">" + tmp)
}

func (buff *Buffer) HandleEachPipe() {
	buff.Pop()
	for _, next := range buff.Next {
		// TODO
		// buff.Print()
		CommandHandler(Client, Message, next, Mentions, *SC, buff)
		buff.Pop()
	}
}

func (buff *Buffer) Clean() {
	var cleaned []string
	for _, next := range buff.Next {
		cleaned = append(cleaned, strings.TrimPrefix(next, "&"))
	}
	buff.Next = cleaned
}

func CommandHandler(client *discordgo.Session, message *discordgo.MessageCreate, content string, mentions []*discordgo.User, sc chan os.Signal, currentBuffer ...*Buffer) {
	// Receive content with mentions stripped
	// Global variabelto use
	// Handle spaces
	content = strings.TrimSpace(content)
	Client = client
	Message = message
	Content = content
	Mentions = mentions
	SC = &sc
	origin := message.ChannelID
	var command string
	content = strings.TrimPrefix(content, command+" ")
	command = strings.Split(content, " ")[0]
	// Check if command has been called from HandleEachPipe or directly form a normal user command
	var buff *Buffer
	if len(currentBuffer) == 0 {
		buff = new(Buffer)
	} else {
		buff = currentBuffer[0]
		// command = buff.Next[0]
	}
	switch strings.ToLower(strings.TrimPrefix(command, "&")) {
	case "ping":
		Ping(buff)
	case "substitute":
		Substitute(buff)
	case "shutdown":
		if config.Config("ID", "Owner") == message.Author.ID {
			Shutdown(buff)
			*SC <- syscall.SIGTERM
		} else {
			buff.Content = "Sorry, I don't think you have enough permissions to use this."
		}

	case "pipe":
		buff.CreateWithPipes(content)
		// Remove pipe itself
		buff.HandleEachPipe()
	case "push":
		buff.Content = RemoveCommand(content)
	case "cat":
		Cat(buff)
	case "cron":
		// Check maximum crons for the user, should be 1 by default
		cronJobs, err := databases.SafeQuery(`select timestamp from jobs where user=?`, message.Author.ID)
		if err != nil {
			Client.ChannelMessageSend(message.ChannelID, "An error occurred while fetching cron jobs.")
			return
		}
		if err != nil {
			Client.ChannelMessageSend(message.ChannelID, "An error occurred while fetching cron jobs.")
			return
		}
		maxCronJobs, err := strconv.Atoi(config.Config("MaxCronJobs", "Default"))
		if err != nil {
			Client.ChannelMessageSend(message.ChannelID, "An error occurred while fetching cron jobs.")
			return
		}
		if len(cronJobs) == maxCronJobs {
			Client.ChannelMessageSend(origin, fmt.Sprintf("You have reached your maximum Cron Job limit. Your next remind is at: %s", cronJobs[0]))
			return
		}
		Client.ChannelMessageSend(origin, "Adding...")
		NewCron(content, buff)
	default:
		return
	}
}
