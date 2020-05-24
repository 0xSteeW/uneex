package commands

import (
	"fmt"
	"io"
	"net/http"
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

func Print(buffer *Buffer) {
	if buffer.Content == "" {
		return
	}
	var commands string
	for _, cmd := range buffer.Pipes {
		commands = commands + cmd + " "
	}
	fieldCommands := &discordgo.MessageEmbedField{Name: "Issued commands", Value: commands}
	fieldResult := &discordgo.MessageEmbedField{Name: ">", Value: buffer.Content}
	fields := []*discordgo.MessageEmbedField{fieldCommands, fieldResult}
	embed := &discordgo.MessageEmbed{Title: "Result:", Fields: fields}
	_, err := Client.ChannelMessageSendEmbed(Message.ChannelID, embed)
	if err != nil {
		Client.ChannelMessageSend(Message.ChannelID, err.Error())
	}
}

func Capitalize(buffer *Buffer) {
	buffer.Content = strings.ToUpper(buffer.Content)
}

func Lower(buffer *Buffer) {
	buffer.Content = strings.ToLower(buffer.Content)
}

func Count(buffer *Buffer) {
	runeArr := []rune(buffer.Content)
	buffer.Content = strconv.Itoa(len(runeArr))
}

func PrintFiles(buffer *Buffer) {
	file := &discordgo.File{Name: "Uneex attachment", Reader: buffer.Files[0]}
	files := []*discordgo.File{file}

	data := &discordgo.MessageSend{Files: files}
	Client.ChannelMessageSendComplex(Message.ChannelID, data)
}

func Substitute(buffer *Buffer, content string) {
	content = RemoveCommand(content)
	// Grab old string until -n
	flags := strings.Split(content, "-n")
	if len(flags) == 1 {
		buffer.Content = "New string wasn't provided"
		return
	}
	// Operator on the left of the split string is the old flag, the right one contains the new and the actual content, so lets split again
	old := strings.TrimSpace(flags[0])
	new := strings.TrimSpace(flags[1])
	buffer.Content = strings.ReplaceAll(buffer.Content, old, new)
}

func Debug(buffer *Buffer) {
	info := fmt.Sprintf("Operations: %d, Errors: %d", len(buffer.Pipes), buffer.Errors)
	buffer.Content = info
}

func Reverse(buffer *Buffer) {
	runeArr := []byte(buffer.Content)
	var newArr []byte
	for i := len(runeArr) - 1; i >= 0; i-- {
		newArr = append(newArr, runeArr[i])
	}
	newString := string(newArr)
	buffer.Content = newString
}

func Cat(buffer *Buffer, content string) {
	buffer.Content = buffer.Content + OnlyRemoveCommand(content)
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

func OnlyRemoveCommand(cmd string) string {
	cmds := strings.Split(cmd, " ")
	return strings.TrimPrefix(cmd, cmds[0])
}

func RemoveCommand(cmd string) string {
	cmds := strings.Split(cmd, " ")
	return strings.TrimSpace(strings.TrimPrefix(cmd, cmds[0]))
}

type Bufferable interface {
	Print()
	Flush()
}

type Buffer struct {
	Content string
	Files   []io.Reader
	Pipes   []string
	Next    []string
	Errors  int
}

func (buff *Buffer) FlushFiles() {
	buff.Files = []io.Reader{}
}

func DownloadToReader(url string) io.Reader {
	response, err := http.Get(url)
	if err != nil {
		return nil
	}
	defer response.Body.Close()

	return response.Body
}

func (buff *Buffer) AttachmentToReader(attachments []*discordgo.MessageAttachment) {
	if attachments == nil {
		return
	}
	buff.FlushFiles()
	for _, attachment := range attachments {
		buff.Files = append(buff.Files, DownloadToReader(attachment.URL))
	}
}

func (buff *Buffer) Pop() {
	buff.Next = buff.Next[1:]
}

func Transfer(origin, destination *Buffer) {
	destination = origin
}

func (buff *Buffer) CreateWithPipes(content string) {
	buff.Pipes = strings.Split(content, "|")
	if len(buff.Pipes) <= 1 {
		buff.Next = []string{content}
	} else {
		buff.Next = buff.Pipes
	}
	buff.Clean()
}

func (buff *Buffer) Print() {
	var tmp string
	for _, p := range buff.Next {
		tmp = tmp + p
	}
}

func (buff *Buffer) HandleEachPipe() {
	// buff.Pop()
	// //FIXME only for &pipe
	maxPipes, _ := strconv.Atoi(config.Config("MaxPipes", "Default"))
	if len(buff.Next) >= maxPipes && Message.Author.ID != config.Config("ID", "Owner") {
		Client.ChannelMessageSend(Message.ChannelID, "Sorry, you've have reached the maximum pipe limit: "+config.Config("MaxPipes", "Default"))
		return
	}
	for _, next := range buff.Next {
		cmds := strings.Split(next, " ")
		if cmds[0] == "&pipe" {
			continue
		}
		CommandHandler(Client, Message, next, Mentions, *SC, buff)
		buff.Pop()
	}
	// By default print if the buffer is on the end
	if buff.Errors != 0 {
		return
	}
	Print(buff)
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
		buff.CreateWithPipes(content)
		buff.HandleEachPipe()
		return
	} else {
		buff = currentBuffer[0]
		// command = buff.Next[0]
	}

	switch strings.ToLower(strings.TrimPrefix(command, "&")) {
	case "ping":
		Ping(buff)
	case "substitute", "replace":
		Substitute(buff, content)
	case "shutdown":
		if config.Config("ID", "Owner") == message.Author.ID {
			Shutdown(buff)
			*SC <- syscall.SIGTERM
		} else {
			buff.Content = "Sorry, I don't think you have enough permissions to use this."
		}
	case "echo":
		buff.Content = RemoveCommand(content)
	case "upper":
		Capitalize(buff)
	case "reverse":
		Reverse(buff)
	case "lower":
		Lower(buff)
	case "grep":
	case "print":
		Print(buff)
	case "wc":
		Count(buff)
	case "debug":
		Debug(buff)
	case "sort":

	case "bold":
		buff.Content = fmt.Sprintf("**%s**", buff.Content)
	case "italic":
		buff.Content = fmt.Sprintf("*%s*", buff.Content)
	case "alternate":

	case "grab", "pick":
		var grab *discordgo.Message
		if RemoveCommand(content) != "" {
			msg, err := Client.ChannelMessage(Message.ChannelID, RemoveCommand(content))
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			grab = msg
		} else {
			previous, err := Client.ChannelMessages(Message.ChannelID, 1, Message.ID, "", "")
			if err != nil {
				return
			}
			grab = previous[0]
		}
		if grab.Attachments != nil {
			if grab.Content != "" {
				buff.Content = grab.Content
			}
			buff.AttachmentToReader(grab.Attachments)
		} else {
			buff.Content = grab.Content
		}
	case "printfiles":
		PrintFiles(buff)
	case "cat":
		Cat(buff, content)
	case "flush":
		buff.Content = ""

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
		buff.Errors += 1
		return
	}
}
