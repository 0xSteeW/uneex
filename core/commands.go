package commands

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
	config "uneex/config"
	databases "uneex/databases"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/gographics/imagick.v3/imagick"
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

// Time functions with time.Now()
func Timer(start time.Time) string {

	duration := time.Since(start)
	return duration.String()
}

func Shutdown(buffer *Buffer) {
	fmt.Println("Shutting down...")
	Client.ChannelMessageSend(Message.ChannelID, "Shutdown requested, proceeding...")
}

func Print(buffer *Buffer) {
	if buffer.Content == "" {
		return
	}
	fieldResult := &discordgo.MessageEmbedField{Name: ">", Value: buffer.Content}
	fields := []*discordgo.MessageEmbedField{fieldResult}
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
	var files []*discordgo.File
	for filename, blob := range buffer.Files {
		reader := strings.NewReader(string(blob))
		file := &discordgo.File{Name: filename, Reader: reader}
		files = append(files, file)
	}

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

func Invert(buffer *Buffer) {
	wand, pw := OpenIM(buffer)
	if wand == nil || pw == nil {
		return
	}
	wand.NegateImage(false)
	SaveBlob(buffer, wand, pw)
}

// Remember to destroy wand after using
func OpenIM(buffer *Buffer) (*imagick.MagickWand, *imagick.PixelWand) {
	var wand *imagick.MagickWand
	var pw *imagick.PixelWand
	for _, blob := range buffer.Files {
		ftype, _ := GetFileType(blob)
		if ftype != "image" {
			buffer.Content = "Provided file wasn't an image."
			buffer.FlushFiles()
			return nil, nil
		}
		wand = imagick.NewMagickWand()
		err := wand.ReadImageBlob(blob)
		if err != nil {
			return nil, nil
		}
		pw = imagick.NewPixelWand()

	}
	return wand, pw
}

func SaveBlob(buffer *Buffer, wand *imagick.MagickWand, pw *imagick.PixelWand) {
	var name string
	for n := range buffer.Files {
		name = n
	}
	var tmp map[string][]byte
	tmp = make(map[string][]byte)
	newFlipped := wand.GetImageBlob()
	defer wand.Destroy()
	defer pw.Destroy()

	tmp[name] = newFlipped
	buffer.Files = tmp
}

func Rotate(buffer *Buffer, content string) {
	// timer := time.Now()
	wand, pw := OpenIM(buffer)
	if wand == nil || pw == nil {
		return
	}
	// ----------------------------------------------------------------------------
	angleRaw := RemoveCommand(content)
	var angle float64
	switch strings.ToLower(angleRaw) {
	case "right":
		angle = 90
	case "left":
		angle = -90
	case "up":
		angle = 180
	case "down":
		angle = -180
	default:
		buffer.Content = "Provided direction is not valid."
		buffer.FlushFiles()
		return
	}
	err := wand.RotateImage(pw, angle)
	if err != nil {
		buffer.Content = "Could not rotate image"
		buffer.FlushFiles()
		return
	}
	// ----------------------------------------------------------------------------
	SaveBlob(buffer, wand, pw)

	// defer Client.ChannelMessageSend(Message.ChannelID, Timer(timer))
}

func Cat(buffer *Buffer, content string) {
	buffer.Content = buffer.Content + OnlyRemoveCommand(content)
}

func Avatar(buffer *Buffer) {
	var avatarFile []byte
	var name string
	var extension string
	if len(Mentions) == 0 {
		avatarFile = DownloadToBytes(Message.Author.AvatarURL(""))
		_, extension = GetFileType(avatarFile)
		name = Message.Author.ID + extension
	} else {
		avatarFile = DownloadToBytes(Mentions[0].AvatarURL(""))
		_, extension := GetFileType(avatarFile)
		name = Mentions[0].ID + extension
	}
	buffer.AddFile(name, avatarFile)
}

func ServerIcon(buffer *Buffer) {
	guild, err := Client.Guild(Message.GuildID)
	if err != nil {
		return
	}
	reader := DownloadToBytes(guild.IconURL())
	_, extension := GetFileType(reader)
	buffer.AddFile(guild.ID+extension, reader)
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
	Files   map[string][]byte
	Pipes   []string
	Next    []string
	Errors  int
}

func (buff *Buffer) FlushFiles() {
	buff.Files = nil
}

func GetFileType(reader []byte) (rawtype string, extension string) {
	by := reader
	ftype := http.DetectContentType(by)
	rawtype = fmt.Sprintf("%s", strings.Split(ftype, "/")[0])
	extension = fmt.Sprintf(".%s", strings.Split(ftype, "/")[1])
	return rawtype, extension
}

func DownloadToBytes(url string) []byte {
	response, err := http.Get(url)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	defer response.Body.Close()
	raw, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil
	}
	return raw

}

func (buff *Buffer) AttachmentToBytes(attachments []*discordgo.MessageAttachment) {
	if attachments == nil {
		return
	}
	var files map[string][]byte
	files = make(map[string][]byte)
	for _, attachment := range attachments {
		downloaded := DownloadToBytes(attachment.URL)
		files[attachment.Filename] = downloaded
	}
	buff.Files = files
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

func (buff *Buffer) AddFile(fname string, reader []byte) {
	file := make(map[string][]byte)
	file[fname] = reader
	buff.Files = file
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
	if buff.Files != nil {
		PrintFiles(buff)
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
		buff.CreateWithPipes(content)
		buff.HandleEachPipe()
		return
	} else {
		buff = currentBuffer[0]
		// command = buff.Next[0]
	}

	switch strings.ToLower(strings.TrimPrefix(command, "&")) {
	case "help":
		Help(buff)
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
	case "avatar", "av":
		Avatar(buff)
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
	case "servericon":
		ServerIcon(buff)
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
			buff.AttachmentToBytes(grab.Attachments)
		} else {
			buff.Content = grab.Content
		}
	case "printfiles":
		PrintFiles(buff)
	case "invert":
		Invert(buff)
	case "rotate":
		Rotate(buff, content)
	case "cat":
		Cat(buff, content)
	case "flush":
		buff.Content = ""
		buff.Files = nil

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

func Help(buffer *Buffer) {
	var fields []*discordgo.MessageEmbedField
	// Text editing commands
	rowHelp := &discordgo.MessageEmbedField{Name: "help", Value: "Help shows a simple help command"}
	rowPing := &discordgo.MessageEmbedField{Name: "ping", Value: "Show a simple RTT menu"}
	rowReplace := &discordgo.MessageEmbedField{Name: "substitute, replace (old -n new)", Value: "Replace string from the buffer content with another of your choosing. To separate old string and new one please use the -n flag"}
	rowShutdown := &discordgo.MessageEmbedField{Name: "shutdown", Value: "Don't you try it..."}
	rowEcho := &discordgo.MessageEmbedField{Name: "echo", Value: "Push a string to the current buffer. It will replace any previous strings on it."}
	rowPrint := &discordgo.MessageEmbedField{Name: "print", Value: "Force to print the content in the buffer before the command ends."}
	rowLower := &discordgo.MessageEmbedField{Name: "lower", Value: "Lower the content in the buffer."}
	rowWc := &discordgo.MessageEmbedField{Name: "wc", Value: "Count characters in the buffer content including spaces."}
	rowDebug := &discordgo.MessageEmbedField{Name: "debug", Value: "Replace buffer with debug info."}
	rowBold := &discordgo.MessageEmbedField{Name: "bold", Value: "Turn text in buffer to bold."}
	rowItalic := &discordgo.MessageEmbedField{Name: "italic", Value: "Turn text in buffer to italic."}
	rowGrab := &discordgo.MessageEmbedField{Name: "grab, pick (messageID)", Value: "Copy the string content of the provided message ID into buffer. If no ID is specified it will copy latest message."}
	rowCat := &discordgo.MessageEmbedField{Name: "cat (string)", Value: "Concatenate current buffer with the provided string."}
	rowFlush := &discordgo.MessageEmbedField{Name: "flush", Value: "Completely empty current buffer, including files."}
	rowCron := &discordgo.MessageEmbedField{Name: "cron", Value: "Work in progress."}
	rowSyntax := &discordgo.MessageEmbedField{Name: "Syntax", Value: "Syntax: &command1 | &command2"}
	rowReverse := &discordgo.MessageEmbedField{Name: "reverse", Value: "Reverses current buffer string."}

	// Image editing commands
	//
	rowAvatar := &discordgo.MessageEmbedField{Name: "avatar, av (mention)", Value: "Save your avatar image to the buffer. If you mention an user, it will be added instead."}
	rowServerIcon := &discordgo.MessageEmbedField{Name: "servericon", Value: "Push the server icon to the buffer."}
	rowPrintFiles := &discordgo.MessageEmbedField{Name: "printfiles", Value: "Force print current files in the buffer."}

	// Append every help row
	fields = []*discordgo.MessageEmbedField{rowSyntax, rowHelp, rowReverse, rowPing, rowReplace, rowShutdown, rowAvatar, rowEcho, rowPrint, rowLower, rowWc, rowServerIcon, rowDebug, rowBold, rowItalic, rowGrab, rowPrintFiles, rowCat, rowFlush, rowCron}

	embed := &discordgo.MessageEmbed{Title: "Help menu", Description: "Current commands. The bot prints the current buffer by default when the command ends.", Fields: fields}
	Client.ChannelMessageSendEmbed(Message.ChannelID, embed)
}
