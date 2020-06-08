package help

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

const introduction = `
Welcome to Uneex discord bot.  I am an advanced bot that resembles the Unix/Linux commands syntax (hence the name Uneex),
providing a way to chain commands and work with previous results as the Linux command line does. I work with things called buffers,
which are no more than temporal storages for every command you issue. These buffers just store Text and Files in them, and by default they are printed
to the current channel when the command ends.
Most commands "push" text into the buffer, instead of printing it directly, so you can work later with it. This happens as well with images.
To make a quick demonstration of how can this bot help you we can see the **addemoji, copy and unemoji** commands.
What copy does is copy all the content from a Message (including files), and works with it in the current buffer. Imagine the user Foo#0001 sent a message
like this: Hello world! :grin:
Imagine the message ID is 123456789. We can copy the message with &copy 123456789 . Now if we just ***pipe*** the current buffer to the **unemoji** command,
we can get that emoji's image!
&copy 123456789 | &unemoji
This would print the image file of the emoji. Commands are separated by | , and every command must be preceded by the prefix &.
Now, lets add more spice, so we will rotate that emoji 90º to the right and we will add it as an emoji to the current server!
&copy 123456789 | &unemoji | &rotate right | &addemoji myemojiname
Et Voilá! We now have a new emoji :myemojiname: !
This is only an example of the countless oportunities this bot offers, so mess around with it and have fun!
I was created by SteeW#7718 , so any doubts or support you can contact him on discord.

You can get specific help of one of these topics by using &help topic
Arguments inside () are mandatory whereas [] means optional.
`

func BasicHelpFields() []*discordgo.MessageEmbedField {

	topics := `
Text Edition: text
Image Edition: image
Moderation: moderation
`
	rowTopics := &discordgo.MessageEmbedField{Name: "Topics (At the right is the topic name for the help command)", Value: topics}
	return []*discordgo.MessageEmbedField{rowTopics}
}

func GenerateHelp(section string) *discordgo.MessageEmbed {
	var fields []*discordgo.MessageEmbedField
	var description string
	switch strings.ToLower(section) {
	case "basic":
		description = introduction
		fields = BasicHelpFields()
	case "text":
		fields = TextEmbedFields()
	case "image":
		fields = ImageEmbedFields()
	case "moderation":
		fields = ModerationEmbedFields()
	default:
		fields = ErrorFields()
	}
	if description != "" {
		return &discordgo.MessageEmbed{Title: section, Fields: fields, Description: description}
	}
	return &discordgo.MessageEmbed{Title: section, Fields: fields}
}

func ErrorFields() []*discordgo.MessageEmbedField {
	rowError := &discordgo.MessageEmbedField{Name: "Error", Value: "The topic you provided is not valid."}
	return []*discordgo.MessageEmbedField{rowError}
}

func TextEmbedFields() []*discordgo.MessageEmbedField {
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
	rowGrab := &discordgo.MessageEmbedField{Name: "copy, grab, pick (messageID)", Value: "Copy the string content of the provided message ID into buffer. If no ID is specified it will copy latest message."}
	rowCat := &discordgo.MessageEmbedField{Name: "cat (string)", Value: "Concatenate current buffer with the provided string."}
	rowFlush := &discordgo.MessageEmbedField{Name: "flush", Value: "Completely empty current buffer, including files."}
	rowCron := &discordgo.MessageEmbedField{Name: "cron", Value: "Work in progress."}
	rowSyntax := &discordgo.MessageEmbedField{Name: "Syntax", Value: "Syntax: &command1 | &command2"}
	rowReverse := &discordgo.MessageEmbedField{Name: "reverse", Value: "Reverses current buffer string."}
	rowUpper := &discordgo.MessageEmbedField{Name: "upper", Value: "Capitalises current buffer."}
	rowB64Encode := &discordgo.MessageEmbedField{Name: "b64encode [Text]", Value: "Base64 encode string in buffer or provided one."}
	rowB64Decode := &discordgo.MessageEmbedField{Name: "b64decode [Text]", Value: "Base64 decode string in buffer or provided one."}
	return []*discordgo.MessageEmbedField{rowBold, rowCat, rowB64Encode, rowB64Decode, rowCron, rowDebug, rowEcho, rowFlush, rowGrab, rowHelp, rowItalic, rowLower, rowPing, rowPrint, rowReplace, rowReverse, rowShutdown, rowSyntax, rowUpper, rowWc}
}

func ModerationEmbedFields() []*discordgo.MessageEmbedField {
	rowKick := &discordgo.MessageEmbedField{Name: "kick (IDS/Mentions) -r [Reason]", Value: ""}
	rowMute := &discordgo.MessageEmbedField{Name: "mute (IDs/Mentions)", Value: ""}
	rowList := &discordgo.MessageEmbedField{Name: "list", Value: "Get all users from the server and save them int the buffer."}
	rowFind := &discordgo.MessageEmbedField{Name: "find `(Regex expression)`", Value: ""}

	rowBan := &discordgo.MessageEmbedField{Name: "ban (IDS/Mentions) -d (Days)", Value: "Ban one or multiple users. Mentions ,ids and users stored with &find or &list are accepted. Use -d [days] to specify ban time."}
	rowCleanSpam := &discordgo.MessageEmbedField{Name: "cleanspam (Max)", Value: "Clean possible spam messages, with a maximum of 500."}
	rowCleanBulk := &discordgo.MessageEmbedField{Name: "cleanbulk (Max)", Value: "Clean all previous messages, with a maximum of 100."}
	rowServerInfo := &discordgo.MessageEmbedField{Name: "serverinfo", Value: "Provide some basic server information."}
	rowNick := &discordgo.MessageEmbedField{Name: "nick [IDS/Mentions] -n (name)", Value: "Rename all mentioned users. It also works with users in buffer. To set the nickname use -n nickname at the end. Use &nick -n RESET to reset user nicknames."}
	rowInfo := &discordgo.MessageEmbedField{Name: "info [Mention]", Value: "Provide some information about mentioned user. Defaults to you if no mentions are provided."}
	return []*discordgo.MessageEmbedField{rowBan, rowCleanBulk, rowCleanSpam, rowMute, rowList, rowFind, rowInfo, rowKick, rowNick, rowServerInfo}
}

func ImageEmbedFields() []*discordgo.MessageEmbedField {
	rowAvatar := &discordgo.MessageEmbedField{Name: "avatar, av [mention]", Value: ""}
	rowServerIcon := &discordgo.MessageEmbedField{Name: "servericon", Value: ""}
	rowPrintFiles := &discordgo.MessageEmbedField{Name: "printfiles", Value: ""}
	rowBlur := &discordgo.MessageEmbedField{Name: "blur (Amount)", Value: ""}
	rowInvert := &discordgo.MessageEmbedField{Name: "invert", Value: ""}
	rowRotate := &discordgo.MessageEmbedField{Name: "rotate (Direction)", Value: ""}
	rowUnemoji := &discordgo.MessageEmbedField{Name: "unemoji [Emoji]", Value: ""}
	rowAddEmoji := &discordgo.MessageEmbedField{Name: "addemoji (Name)", Value: ""}
	rowDeleteEmoji := &discordgo.MessageEmbedField{Name: "deleteemoji (Name)", Value: ""}
	return []*discordgo.MessageEmbedField{rowAddEmoji, rowAvatar, rowBlur, rowInvert, rowDeleteEmoji, rowPrintFiles, rowRotate, rowServerIcon, rowUnemoji}
}

func GetCommandHelp(command string) (explanation string, usage string) {
	if help, ok := commandsHelp[command]; ok {
		return help, ""
	}
	return "", ""
}

var commandsHelp map[string]string = map[string]string{
	"avatar":     "Save your avatar image to the buffer. If you mention an user, it will be added instead.",
	"servericon": "Push the server icon to the buffer.",
	"printfiles": "Force print current files in the buffer.",
	"blur":       "Blur images on buffer, with a maximum amount of 50.",
	"invert":     "Convert images on buffer to negative.",
	"rotate":     "Rotate images on buffer. Valid directions: up, down, left, right.",
	"unemoji":    "Get downloadable image of an emoji. It can also get emojis from copied messages. Pushes images to the buffer. Caution: This only works with custom emojis!",
	"emoji":      "Add or delete specified emojis or emojis on buffer.",

	"kick":                                 "Kick one or multiple users. Mentions ,ids and users stored with &find or &list are accepted. Use -r [reason] at the end to give a reason.",
	"mute":                                 "Mute a user (work in progress)",
	"find":                                 "Filter any category with the given expression. Categories available are: channels, messages, users. Example: &find `^A` -t messages (Will find every message from the last 100 that starts with A.)",
	"ban (Mentions/IDS) -r Reason -d Days": "Ban selected users. (Will grab users on buffer if no mention or ID was provided).",

	"delete --type Type": "Delete every item of the selected category. It must be preceded by a &find command. Example: &find `^[0-9]+` -t users | &delete -t messages (Will delete every message starting with a number from the last 100 messages.)",
}
