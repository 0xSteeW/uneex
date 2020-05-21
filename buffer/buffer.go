package buffer

import (
	"github.com/bwmarrin/discordgo"
)

type Bufferable interface {
	Signal()
	Transfer(*Buffer)
}

type Buffer struct {
	Content string
	Message *discordgo.MessageCreate
	Session *discordgo.Session
	Next    bool
}

func (buff *Buffer) Transfer(destination *Buffer) {
	destination = buff
}

func (buff *Buffer) Signal() {
	buff.Session.ChannelMessageSend(buff.Message.ChannelID, buff.Content)
}

func (buff *Buffer) HandleEachPipe() {
	for buff.Next {

	}
}
