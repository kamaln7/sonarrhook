package config

import (
	"log"
	"os"

	"github.com/BurntSushi/toml"
)

type Obj struct {
	HTTP     HTTPConfig
	Mailgun  MailgunConfig
	Contacts Contacts
	Series   map[string]([]string)
}

type HTTPConfig struct {
	Host, Key string
	Port      int
}

type MailgunConfig struct {
	From, Domain, APIKey, PublicAPIKey string
}

type Contacts map[string]string

func Read() Obj {
	configfile := "./config/config.toml"
	_, err := os.Stat(configfile)
	if err != nil {
		log.Fatal("config file is missing: ", configfile)
	}

	var config Obj
	if _, err := toml.DecodeFile(configfile, &config); err != nil {
		log.Fatal(err)
	}

	return config
}
