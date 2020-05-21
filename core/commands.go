package commands

import (
	"fmt"
	"strconv"
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
	embed := &discordgo.MessageEmbed{Title: "Pong!", Description: "Successfull ping! Showing RTT:", Fields: fields}
	es.Client.ChannelMessageSendEmbed(es.Message.ChannelID, embed)
}

func (es *ExportedSession) Shutdown() {
	fmt.Println("Shutting down...")
	es.Client.ChannelMessageSend(es.Message.ChannelID, "Shutdown requested, proceeding...")
}

func (es *ExportedSession) NewCron() {
	databases.SafeExec(`update table users values`)
}
