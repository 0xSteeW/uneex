package cron

import (
	"fmt"
	"time"

	databases "uneex/databases"

	"github.com/jonas747/dshardmanager"
)

type RowContent struct {
	Timestamp time.Time
	User      string
	Content   string
}

func Worker(stop chan bool, manager *dshardmanager.Manager) {
	for {
		select {
		case <-stop:
			break
		default:
			//databases.SafeQuery(date, `select TIME_FORMAT(TIME(CONVERT_TZ(SYSDATE(),'+02:00','+00:00')), '%H:%i')`)
			remindUsers, err := databases.Database.Query(`select * from jobs where TIME_FORMAT(TIME(timestamp), '%H:%i')=TIME_FORMAT(TIME(?), '%H:%i')`, time.Now().UTC())
			if err != nil {
				fmt.Println("[Worker]:", err)
				return
			}
			defer remindUsers.Close()
			var users []*RowContent
			for remindUsers.Next() {
				scan := new(RowContent)
				err := remindUsers.Scan(&scan.Timestamp, &scan.User, &scan.Content)
				if err != nil {
					continue
				}
				users = append(users, scan)

			}
			go messageQueued(users, manager)
			go deleteOlder()
			time.Sleep(1 * time.Minute)
		}
	}
}

func messageQueued(rc []*RowContent, manager *dshardmanager.Manager) {
	dmsession := manager.Session(0)
	for _, userLoop := range rc {
		dm, err := dmsession.UserChannelCreate(userLoop.User)
		if err != nil {
			continue
		}
		dmsession.ChannelMessageSend(dm.ID, "You have a reminder: "+userLoop.Content)
		time.Sleep(1 * time.Second)
	}
}

func deleteOlder() {
	databases.SafeExec(`delete from jobs where timestamp<?`, time.Now().UTC())
}
