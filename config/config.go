package config

import (
	"sync"

	"github.com/go-ini/ini"
)

var Conf *ini.File
var mutex = &sync.Mutex{}

func Config(name string, section ...string) string {
	mutex.Lock()
	defer mutex.Unlock()
	if len(section) == 0 {
		return Conf.Section("").Key(name).String()
	}
	return Conf.Section(section[0]).Key(name).String()
}
