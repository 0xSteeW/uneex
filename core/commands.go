package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	databases "uneex/databases"

	"github.com/bwmarrin/discordgo"
)

// Global core commands
//
//
type ExportedSession struct {
	Client  *discordgo.Session
	Message *discordgo.MessageCreate
}

func (es *ExportedSession) Ping() {
	latency := es.Client.HeartbeatLatency()
	var fields []*discordgo.MessageEmbedField
	fieldPingNormal := &discordgo.MessageEmbedField{Name: "Truncated", Value: latency.String()}
	fieldPingNanoseconds := &discordgo.MessageEmbedField{Name: "Nanoseconds", Value: strconv.FormatInt(latency.Nanoseconds(), 10)}
	fieldPingMicroseconds := &discordgo.MessageEmbedField{Name: "Microseconds", Value: strconv.FormatInt(latency.Microseconds(), 10)}
	fields = []*discordgo.MessageEmbedField{fieldPingNormal, fieldPingNanoseconds, fieldPingMicroseconds}
	embed := &discordgo.MessageEmbed{Title: "Pong!", Description: "Successful ping! Showing RTT:", Fields: fields}
	es.Client.ChannelMessageSendEmbed(es.Message.ChannelID, embed)
}

func (es *ExportedSession) Shutdown() {
	fmt.Println("Shutting down...")
	es.Client.ChannelMessageSend(es.Message.ChannelID, "Shutdown requested, proceeding...")
}

func (es *ExportedSession) Cat(buffer *Buffer) {
	es.Client.ChannelMessageSend(es.Message.ChannelID, buffer.Content)
}

func (es *ExportedSession) NewCron(content string) {
	timeStamp := strings.Split(content, " -c")[0]
	es.Client.ChannelMessageSend(es.Message.ChannelID, "`"+timeStamp+"`")
	timeStampParse, err := time.Parse(`Mon Jan 2 15:04:05 -0700 MST 2006`, timeStamp)
	if err != nil {
		es.Client.ChannelMessageSend(es.Message.ChannelID, "Sorry, the time format you provided isn't valid.")
		es.Client.ChannelMessageSend(es.Message.ChannelID, err.Error())
		return
	}
	remind := strings.Split(content, "-c")[1:]
	var remindLiteral string
	for _, word := range remind {
		remindLiteral = remindLiteral + word
	}
	userAlreadyExists, err := databases.SafeQuery(`select * from user where id=?`, es.Message.Author.ID)
	if err != nil {
		return
	}
	if len(userAlreadyExists) == 0 {
		databases.SafeExec(`insert into user values(?)`, es.Message.Author.ID)
	}
	databases.SafeExec(`insert into jobs values(?,?,?)`, timeStampParse, es.Message.Author.ID, remindLiteral)
	es.Client.ChannelMessageSend(es.Message.ChannelID, "Succesfully added remind for "+timeStampParse.String()+" with content: "+remindLiteral)
}

type Bufferable interface {
	Signal()
	Transfer(*Buffer)
}

type Buffer struct {
	Content string
	Pipes   []string
	Message *discordgo.MessageCreate
	Session *discordgo.Session
}

func Transfer(origin, destination *Buffer) {
	destination = origin
}

func (buff *Buffer) HandleEachPipe(es ExportedSession) {
	for _, pipe := range buff.Pipes {
		// TODO
		pipe = pipe
		buff.Content = "test"
	}
	es.Cat(buff)
}
