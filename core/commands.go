package commands

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
	config "uneex/config"
	databases "uneex/databases"
	help "uneex/help"
	"uneex/moderation"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dshardmanager"
	"gopkg.in/gographics/imagick.v3/imagick"
)

// Global core commands
//
//
var Client *discordgo.Session
var Message *discordgo.MessageCreate
var Content string
var Mentions []*discordgo.User
var Manager *dshardmanager.Manager
var SC *chan os.Signal

func ThisShard(buffer *Buffer) {
	ID, _ := strconv.Atoi(Message.GuildID)
	shard := Manager.SessionForGuild(int64(ID))
	buffer.Content = strconv.Itoa(shard.ShardID)
}

func Ping(buffer *Buffer) {
	latency := Client.HeartbeatLatency()
	var fields []*discordgo.MessageEmbedField
	fieldPingNormal := &discordgo.MessageEmbedField{Name: "Truncated", Value: latency.String()}
	fieldPingNanoseconds := &discordgo.MessageEmbedField{Name: "Nanoseconds", Value: strconv.FormatInt(latency.Nanoseconds(), 10)}
	fieldPingMicroseconds := &discordgo.MessageEmbedField{Name: "Microseconds", Value: strconv.FormatInt(latency.Microseconds(), 10)}
	fieldPingShards := &discordgo.MessageEmbedField{Name: "Shards", Value: strconv.Itoa(Client.ShardCount)}
	fields = []*discordgo.MessageEmbedField{fieldPingNormal, fieldPingNanoseconds, fieldPingMicroseconds, fieldPingShards}
	var color int
	var description string
	if latency.Microseconds() > 200000 {
		color = 0xff0000
		description = "Uh oh, something's wrong with the ping..."
	} else {
		color = 0x00ff00
		description = "Successful ping!"
	}
	embed := &discordgo.MessageEmbed{Title: "Pong!", Description: description, Fields: fields, Color: color}

	Client.ChannelMessageSendEmbed(Message.ChannelID, embed)
}

// Crypto section
func Base64Decode(buffer *Buffer, content string) {
	content = RemoveCommand(content)
	var base string
	if content == "" {
		if buffer.Content != "" {
			base = buffer.Content
		} else {
			buffer.Content = "You didnt provide any string"
			return
		}
	} else {
		base = content
	}
	decoded, err := base64.StdEncoding.DecodeString(base)
	if err != nil {
		buffer.Content = "Could not decode string."
		return
	}
	buffer.Content = string(decoded)
}

func Base64Encode(buffer *Buffer, content string) {
	content = RemoveCommand(content)
	var base string
	if content == "" {
		if buffer.Content != "" {
			base = buffer.Content
		} else {
			buffer.Content = "You didnt provide any string"
			return
		}
	} else {
		base = content
	}
	buffer.Content = base64.StdEncoding.EncodeToString([]byte(base))
}

func GetMentions(content string) []*discordgo.User {
	var userList []*discordgo.User
	detectMention := regexp.MustCompile(`<@![0-9]{18}>`)
	params := strings.Split(content, " ")
	if len(params) == 0 {
		return nil
	}
	for _, userid := range params {
		userid = strings.TrimSpace(userid)
		if len(userid) == 18 {
			user, err := Client.User(userid)
			if err == nil {
				userList = append(userList, user)
			}
		} else if detectMention.MatchString(userid) {
			name := userid[3 : len(userid)-1]
			user, err := Client.User(name)
			if err == nil {
				userList = append(userList, user)
			}
		}
	}
	return userList
}

// Time functions with time.Now()
func Timer(start time.Time) string {

	duration := time.Since(start)
	return duration.String()
}

func ConcatRoleSlice(slice []string, guild *discordgo.Guild) string {
	var roleMap map[string]string
	var cat string
	roleMap = make(map[string]string)
	for _, role := range guild.Roles {
		roleMap[role.ID] = role.Name
	}
	for _, element := range slice {
		if role, ok := roleMap[element]; ok {
			cat = cat + "" + role + " "
		}
	}
	return cat
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

// Moderation
func GetPermissionsInt() int {
	perms, err := Client.UserChannelPermissions(Message.Author.ID, Message.ChannelID)
	if err != nil {
		return 0
	}
	return perms
}

func StripMentions(content string) (params []string, mentionSlice []string) {
	content = RemoveCommand(content)
	listOfParams := strings.Split(content, " ")
	mention := regexp.MustCompile(`<@![0-9]{18}>`)
	if len(listOfParams) == 0 {
		return nil, nil
	}
	var newParams []string
	var mentions []string
	for _, param := range listOfParams {
		if mention.Match([]byte(param)) {
			mentions = append(mentions, param)
		} else {
			newParams = append(newParams, param)
		}
	}
	return newParams, mentions
}

func FormatSliceToString(slice []string) string {
	if slice == nil {
		return ""
	}
	var final string
	for _, el := range slice {
		final = final + el
	}
	return strings.TrimSpace(final)
}

func Kick(buffer *Buffer, content string) {
	if moderation.HasPermission("kick", GetPermissionsInt()) {
		params := strings.Split(RemoveCommand(content), "-r")
		var reason string
		var paramsLeft string
		if len(params) < 2 {
			buffer.Content = "No reason was provided, proceeding."
			paramsLeft = RemoveCommand(content)
		} else {
			reason = params[1]
			paramsLeft = params[0]
		}
		mentions := GetMentions(paramsLeft)
		if len(mentions) > 0 {
			var err error
			for _, userToKick := range mentions {
				if reason != "" {
					err = Client.GuildMemberDeleteWithReason(Message.GuildID, userToKick.ID, reason)
				} else {
					err = Client.GuildMemberDelete(Message.GuildID, userToKick.ID)
				}
			}
			if err != nil {
				buffer.Content = "One or more users could not be kicked."
			} else {
				buffer.Content = "Successfully kicked all users."
			}
		} else {
			buffer.Content = "You didn't provide any user."
		}
	} else {
		buffer.Content = "Sorry, you don't have permission for this."
	}
}

func Delete(buffer *Buffer, content string) {
	if !moderation.HasPermission("manageMessages", GetPermissionsInt()) {
		buffer.Content = "Sorry, you don't have enough permissions."
		return
	}
	if RemoveCommand(content) == "" {
		buffer.Content = "Number of messages hasn't been provided."
		return
	}
	number, err := strconv.Atoi(RemoveCommand(content))
	if err != nil || number > 100 || number <= 0 {
		buffer.Content = "Provided number is too high (100) or not valid"
		return
	}
	messages, err := Client.ChannelMessages(Message.ChannelID, number, "", "", "")
	messageIDS := MessagesToString(messages)
	err = Client.ChannelMessagesBulkDelete(Message.ChannelID, messageIDS)
	if err != nil {
		buffer.Content = "Couldn't delete messages."
		return
	}
	buffer.Content = "Messages deleted successfully."

}

func MessagesToString(messages []*discordgo.Message) []string {
	var final []string
	if messages == nil {
		return nil
	}
	for _, message := range messages {
		final = append(final, message.ID)
	}
	return final
}

func CleanSpam(buffer *Buffer, content string) {
	if !moderation.HasPermission("manageMessages", GetPermissionsInt()) {
		buffer.Content = "You don't have permission for this."
		return
	}
	MAX_RAW := config.Config("MaxSpamMessages", "Default")
	MAX, _ := strconv.Atoi(MAX_RAW)
	numberOfMessages, err := strconv.Atoi(RemoveCommand(content))
	if err != nil {
		buffer.Content = "Number of provided messages is invalid."
		return
	}
	if numberOfMessages > MAX || numberOfMessages <= 0 {
		buffer.Content = "You have exceeded the maximum number of messages: " + strconv.Itoa(MAX) + " or a number lower than 1"
		return
	}
	// Get messages up to numberOfMessages
	var left int
	var rounds int
	if numberOfMessages < 100 {
		left = numberOfMessages
	} else {
		rounds = int(math.Trunc(float64(numberOfMessages / 100)))
		left = numberOfMessages - rounds
	}

	var lastMessageID string
	var messageList []*discordgo.Message
	i := 0
	for i < rounds {
		messages, err := Client.ChannelMessages(Message.ChannelID, 100, lastMessageID, "", "")
		if err != nil {
			buffer.Content = "Could not retrieve messages"
			return
		}
		messageList = append(messageList, messages...)
		if len(messages) > 0 {
			lastMessageID = messages[len(messages)-1].ID
		}
		i++
	}
	// Append remaining messages
	messages, err := Client.ChannelMessages(Message.ChannelID, left, lastMessageID, "", "")
	messageList = append(messageList, messages...)

	BulkDelete(buffer, FindSpam(messageList))
}

func CleanEmpty(messages []*discordgo.Message) []*discordgo.Message {
	var final []*discordgo.Message
	for _, msg := range messages {
		if msg.Content != "" {
			final = append(final, msg)
		}
	}
	return final
}

func IsSpam(content string) bool {
	// Short Characters finder
	// TODO change this to a database
	const spamRatio = 0.7
	content = strings.TrimSpace(content)
	shortCount := 0
	wordList := strings.Split(content, " ")
	// Regex for multi word messages
	if wordList != nil {
		for _, word := range wordList {
			if len(word) < 3 || len(word) > 10 {
				shortCount += 1
			}
		}
		if float64(shortCount/len(wordList)) > spamRatio {
			return true
		}
	}
	return false
}

func FindSpam(messages []*discordgo.Message) []*discordgo.Message {
	messages = CleanEmpty(messages)
	var final []*discordgo.Message
	for _, message := range messages {
		if IsSpam(message.Content) {
			final = append(final, message)
		}
	}
	return final
}

func BulkDelete(buffer *Buffer, messages []*discordgo.Message) {
	var err error
	var rounds int
	var left int

	rounds = int(math.Trunc(float64(len(messages) / 100)))
	left = len(messages) - (rounds * 100)

	if len(messages) < 100 {
		left = len(messages)
		rounds = -1
	}
	Client.ChannelTyping(Message.ChannelID)
	for i := 0; i <= rounds; i++ {
		err = Client.ChannelMessagesBulkDelete(Message.ChannelID, MessagesToString(messages[i*100:(i+1)*100]))
	}
	if left > 0 {
		err = Client.ChannelMessagesBulkDelete(Message.ChannelID, MessagesToString(messages[:left]))
	}
	if err != nil {
		buffer.Content = "Some messages could not be deleted."
	}
	buffer.Content = "Total correctly removed messages: " + strconv.Itoa(len(messages))
}

// Ban
func Ban(buffer *Buffer, content string) {
	if moderation.HasPermission("ban", GetPermissionsInt()) {
		params := strings.Split(RemoveCommand(content), "-r")
		var reason string
		var paramsLeft string
		if len(params) < 2 {
			buffer.Content = "No reason was provided, proceeding."
			paramsLeft = RemoveCommand(content)
		} else {
			reason = params[1]
			paramsLeft = params[0]
		}
		days := strings.Split(content, "-d")
		if len(days) == 0 {
			buffer.Content = "No amount of days were provided"
			return
		}
		daysInt, err := strconv.Atoi(strings.TrimSpace(days[1]))
		if err != nil || daysInt <= 0 {
			buffer.Content = "Number of days were not valid."
			return
		}
		mentions := GetMentions(paramsLeft)
		if len(mentions) > 0 {
			var err error
			for _, userToKick := range Mentions {
				if reason != "" {
					err = Client.GuildBanCreateWithReason(Message.GuildID, userToKick.ID, reason, daysInt)
				} else {
					err = Client.GuildBanCreate(Message.GuildID, userToKick.ID, daysInt)
				}
			}
			if err != nil {
				buffer.Content = "One or more users could not be banned."
			} else {
				buffer.Content = "Successfully banned all users."
			}
		} else {
			buffer.Content = "You didn't provide any user."
		}
	} else {
		buffer.Content = "Sorry, you don't have permission for this."
	}
}

// TODO
func Info(buffer *Buffer, content string) {
	// Parse user
	//
	var user *discordgo.User
	var joinedTime time.Time

	if RemoveCommand(content) == "" {
		user = Message.Author
	} else {
		user = GetMentions(RemoveCommand(content))[0]
	}
	userField := &discordgo.MessageEmbedField{Name: "User", Value: user.String(), Inline: true}
	guild, err := Client.Guild(Message.GuildID)
	if err != nil {
		return
	}
	var joinedAt *discordgo.Timestamp
	var rolesRaw []string
	for _, member := range guild.Members {
		if member.User.ID == user.ID {
			joinedAt = &member.JoinedAt
			rolesRaw = member.Roles
			break
		}
	}
	if joinedAt == nil {
		return
	}
	joinedTime, err = joinedAt.Parse()
	if err != nil {
		return
	}
	joinDate := &discordgo.MessageEmbedField{Name: "Server Join Date", Value: joinedTime.String(), Inline: true}
	cat := ConcatRoleSlice(rolesRaw, guild)
	if cat == "" {
		cat = "None"
	}
	guildRoles := &discordgo.MessageEmbedField{Name: "User Roles", Value: cat, Inline: false}
	userID := &discordgo.MessageEmbedField{Name: "User ID", Value: user.ID, Inline: false}

	fields := []*discordgo.MessageEmbedField{userField, joinDate, guildRoles, userID}
	embed := &discordgo.MessageEmbed{Title: "User information", Fields: fields}
	_, err = Client.ChannelMessageSendEmbed(Message.ChannelID, embed)
	if err != nil {
		buffer.Content = "Couldn't send info"
		fmt.Println(err.Error())
	}
}

// TODO
func ServerInfo(buffer *Buffer) {
	var guild *discordgo.Guild
	var err error
	guild, err = Client.Guild(Message.GuildID)
	if err != nil {
		return
	}
	guildName := &discordgo.MessageEmbedField{Name: "Server Name", Value: guild.Name, Inline: true}
	guildMembers := &discordgo.MessageEmbedField{Name: "Members", Value: strconv.Itoa(guild.MemberCount), Inline: true}
	guildIcon := &discordgo.MessageEmbedThumbnail{URL: guild.IconURL()}

	fields := []*discordgo.MessageEmbedField{guildName, guildMembers}
	embed := &discordgo.MessageEmbed{Title: "Server information", Fields: fields, Thumbnail: guildIcon}
	_, err = Client.ChannelMessageSendEmbed(Message.ChannelID, embed)
	if err != nil {
		buffer.Content = "Couldn't send info"
		fmt.Println(err.Error())
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

func DeleteEmoji(buffer *Buffer, content string) {
	if !moderation.HasPermission("manageEmojis", GetPermissionsInt()) {
		buffer.Content = "You don't have permission for this."
		return
	}
	content = RemoveCommand(content)
	if content == "" {
		buffer.Content = "You didn't provide an emoji"
		return
	}
	// Get emoji id
	guild, _ := Client.Guild(Message.GuildID)
	emojis := guild.Emojis
	var ID string
	for _, emoji := range emojis {
		if emoji.Name == strings.TrimSpace(content) {
			ID = emoji.ID
		}
	}
	err := Client.GuildEmojiDelete(Message.GuildID, ID)
	if err != nil {
		buffer.Content = "Could not find or delete emoji."
		return
	}
	buffer.Content = "Emoji successfully deleted."
}

func AddEmoji(buffer *Buffer, content string) {
	if !moderation.HasPermission("manageEmojis", GetPermissionsInt()) {
		buffer.Content = "You don't have enough permissions for this."
		buffer.FlushFiles()
		return
	}
	if RemoveCommand(content) == "" {
		buffer.Content = "Emoji name not provided."
		return
	}
	for _, img := range buffer.Files {
		ftype, _ := GetFileType(img)
		if ftype == "image" {
			fullTypeToEncode := http.DetectContentType(img)
			base64image := base64.StdEncoding.EncodeToString(img)
			dataURI := fmt.Sprintf(`data:%s;base64,%s`, fullTypeToEncode, base64image)
			_, err := Client.GuildEmojiCreate(Message.GuildID, RemoveCommand(content), dataURI, nil)
			if err != nil {
				buffer.Content = "Couldn't add emoji. " + err.Error()
				fmt.Println(err.Error())
				return
			}
			buffer.Content = "Successfully added emoji."
			return
		} else {
			buffer.Content = "Provided files weren't images."
		}
	}
	buffer.Content = "No files provided."
}

func Unemoji(buffer *Buffer, content string) {
	isEmoji := regexp.MustCompile(`\ ?<\:.*:[0-9]{18}>\ ?`)
	var searchFor string
	if RemoveCommand(content) != "" {
		searchFor = RemoveCommand(content)
	} else {
		if buffer.Content != "" {
			searchFor = strings.TrimSpace(buffer.Content)
		}
		// Find emojis here to work with them
		items := strings.Split(searchFor, " ")
		if len(items) != 0 {
			for _, item := range items {
				if isEmoji.Match([]byte(item)) {
					searchFor = item
					break
				}
			}
		}
	}
	emojiLink := "https://cdn.discordapp.com/emojis/"
	if isEmoji.Match([]byte(searchFor)) {
		code := strings.Split(searchFor, ":")
		if len(code) < 3 {
			buffer.Content = "Emoji not valid"
			buffer.FlushFiles()
			return
		}
		codeRaw := strings.TrimSuffix(code[2], ">")
		dwurl := emojiLink + codeRaw + ".png"
		emoji := DownloadToBytes(dwurl)
		ftype, ext := GetFileType(emoji)
		if ftype != "image" {
			buffer.Content = "Emoji couldn't be de-emoji'd"
			buffer.FlushFiles()
			return
		}
		buffer.AddFile(codeRaw+ext, emoji)
	}
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

func Blur(buffer *Buffer, content string) {
	wand, pw := OpenIM(buffer)
	MAX_SIGMA_RAW := config.Config("MaxSigma", "Default")
	MAX_SIGMA, _ := strconv.Atoi(MAX_SIGMA_RAW)
	if wand == nil || pw == nil {
		return
	}
	sigma, err := strconv.Atoi(RemoveCommand(content))
	if sigma > MAX_SIGMA || sigma <= 0 {
		buffer.Content = "Sigma is too high. (MAXVAL: )" + strconv.Itoa(MAX_SIGMA)
		buffer.FlushFiles()
		return
	}
	if err != nil {
		buffer.Content = "Provided sigma wasn't a number or it was not valid"
		buffer.FlushFiles()
		return
	}
	err = wand.BlurImage(0, float64(sigma))
	if err != nil {
		buffer.Content = "Could not blur image."
		buffer.FlushFiles()
		return
	}
	SaveBlob(buffer, wand, pw)
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
	Client.ChannelTyping(Message.ChannelID)
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

func Avatar(buffer *Buffer, content string) {
	content = RemoveCommand(content)
	var avatarFile []byte
	var name string
	var extension string
	mention := GetMentions(content)
	if len(mention) == 0 {
		avatarFile = DownloadToBytes(Message.Author.AvatarURL(""))
		_, extension = GetFileType(avatarFile)
		name = Message.Author.ID + extension
	} else {
		avatarFile = DownloadToBytes(mention[0].AvatarURL(""))
		_, extension := GetFileType(avatarFile)
		name = mention[0].ID + extension
	}
	buffer.AddFile(name, avatarFile)
}

func Nick(buffer *Buffer, content string) {
	content = RemoveCommand(content)
	params := strings.Split(content, "-n")
	if len(params) <= 1 {
		buffer.Content = "Nickname or mentions not provided."
		return
	}
	nick := strings.TrimSpace(params[1])
	mentions := GetMentions(content)
	var err error
	var count int
	total := len(mentions)
	for _, user := range mentions {
		err = Client.GuildMemberNickname(Message.GuildID, user.ID, nick)
		if err == nil {
			count += 1
		}
	}
	if err != nil {
		buffer.Content = "Some users could not be nicknamed."
		return
	}
	buffer.Content = "Successfully renamed " + strconv.Itoa(count) + "/" + strconv.Itoa(total) + " mentioned users to: " + nick
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
		CommandHandler(Client, Message, next, Mentions, *SC, Manager, buff)
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

func CommandHandler(client *discordgo.Session, message *discordgo.MessageCreate, content string, mentions []*discordgo.User, sc chan os.Signal, manager *dshardmanager.Manager, currentBuffer ...*Buffer) {
	// Receive content with mentions stripped
	// Global variabelto use
	// Handle spaces
	content = strings.TrimSpace(content)
	Client = client
	Message = message
	Content = content
	Manager = manager
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
		Help(buff, content)
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
		Avatar(buff, content)
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

	case "grab", "pick", "copy":
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
	case "kick":
		Kick(buff, content)
	case "ban":
		Ban(buff, content)
	case "printfiles":
		PrintFiles(buff)
	case "info":
		Info(buff, content)
	case "cleanspam":
		CleanSpam(buff, content)
	case "cleanbulk":
		Delete(buff, content)
	case "serverinfo":
		ServerInfo(buff)
	case "blur":
		Blur(buff, content)
	case "nick":
		Nick(buff, content)
	case "invert":
		Invert(buff)
	case "b64encode":
		Base64Encode(buff, content)
	case "b64decode":
		Base64Decode(buff, content)
	case "shardid":
		ThisShard(buff)
	case "rotate":
		Rotate(buff, content)
	case "cat":
		Cat(buff, content)
	case "unemoji":
		Unemoji(buff, content)
	case "addemoji":
		AddEmoji(buff, content)
	case "deleteemoji":
		DeleteEmoji(buff, content)
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

func Help(buffer *Buffer, content string) {
	var topic string
	if RemoveCommand(content) == "" {
		topic = "basic"
	} else {
		topic = RemoveCommand(content)
	}
	helpEmbed := help.GenerateHelp(topic)
	_, err := Client.ChannelMessageSendEmbed(Message.ChannelID, helpEmbed)
	if err != nil {
		buffer.Content = "Could not send help. " + err.Error()
	}
}
