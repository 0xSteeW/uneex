package commands

import (
	"fmt"

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
	es.Client.ChannelMessageSend(es.Message.ChannelID, es.Client.HeartbeatLatency().String())
}

func (es *ExportedSession) Shutdown() {
	fmt.Println("Shutting down...")
	es.Client.ChannelMessageSend(es.Message.ChannelID, "Shutdown requested, proceeding...")
}
