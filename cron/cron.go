package cron

import (
	"fmt"
	"time"
	databases "uneex/databases"

	"github.com/bwmarrin/discordgo"
)

type RowContent struct {
	Timestamp time.Time
	User      string
	Content   string
}

func Worker(stop chan bool, client *discordgo.Session) {
	for {
		select {
		case <-stop:
			break
		default:
			fmt.Println(time.Now().String())
			remindUsers, err := databases.Database.Query(`select * from jobs where timestamp=current_timestamp()`)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			var users []RowContent
			defer remindUsers.Close()
			for remindUsers.Next() {
				parse := new(RowContent)
				err := remindUsers.Scan(&parse)
				if err != nil {
					continue
				}
				users = append(users, *parse)
			}
			for _, userLoop := range users {
				dm, err := client.UserChannelCreate(userLoop.User)
				if err != nil {
					continue
				}
				client.ChannelMessageSend(dm.ID, "I remind you: "+userLoop.Content)
			}
			time.Sleep(1 * time.Minute)
		}
	}
}
