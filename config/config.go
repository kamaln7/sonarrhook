package config

import (
	"log"
	"os"

	"github.com/BurntSushi/toml"
)

// Obj is a struct containing the different config
// options for sonarrhook
type Obj struct {
	HTTP     HTTPConfig
	Mailgun  MailgunConfig
	Contacts Contacts
	Series   map[string]([]string)
}

// HTTPConfig contains the HTTP server config
type HTTPConfig struct {
	Host, Key string
	Port      int
}

// MailgunConfig contains the Mailgun API and Email configs
type MailgunConfig struct {
	From, Domain, APIKey, PublicAPIKey string
}

// Contacts is a list of contacts and their emails
type Contacts map[string]string

// Read reads the config file and returns a config Obj
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
