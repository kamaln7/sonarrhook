package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/kamaln7/sonarrhook/config"

	"gopkg.in/mailgun/mailgun-go.v1"
)

var (
	Config  config.Obj
	Mailgun mailgun.Mailgun
)

func main() {
	log.Print("reading config file")
	Config = config.Read()

	http.HandleFunc("/download", Download)

	Mailgun = mailgun.NewMailgun(Config.Mailgun.Domain, Config.Mailgun.APIKey, Config.Mailgun.PublicAPIKey)

	listenAddr := fmt.Sprintf("%s:%d", Config.HTTP.Host, Config.HTTP.Port)
	log.Printf("listening on %s", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

type DownloadEvent struct {
	EventType string
	Series    Series
	Episodes  []Episode
}

type Series struct {
	Id          int
	Title, Path string
	TvdbId      uint64
}

type Episode struct {
	Id                                                           int
	EpisodeNumber, SeasonNumber, QualityVersion                  int
	Title, AirDate, AirDateUtc, Quality, ReleaseGroup, SceneName string
}

func Download(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)

		return
	}

	if apiKey := r.URL.Query().Get("key"); apiKey != Config.HTTP.Key {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)

		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Print(err)

		return
	}

	var event DownloadEvent
	err = json.Unmarshal(body, &event)
	if err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)

		return
	}

	if event.EventType != "Download" {
		if event.EventType == "Test" {
			log.Printf("received webhook test request")
			fmt.Fprint(w, "Tested")
		} else {
			log.Print("expected EventType to be \"Download\"")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}

		return
	}

	log.Printf("got series [%d] %s", event.Series.Id, event.Series.Title)

	var contactNames, recipients []string

	contactNames, ok := Config.Series[strconv.Itoa(event.Series.Id)]
	if !ok {
		log.Printf("series id %d not in config", event.Series.Id)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)

		return
	}

	for _, name := range contactNames {
		if _, ok := Config.Contacts[name]; !ok {
			log.Printf("contact %s not in config", name)

			continue
		}

		recipients = append(recipients, Config.Contacts[name])
	}

	if len(recipients) == 0 {
		log.Print("no contacts to notify")

		return
	}

	title := fmt.Sprintf("New %s episodes downloaded", event.Series.Title)
	var message bytes.Buffer

	for _, ep := range event.Episodes {
		message.WriteString(fmt.Sprintf("+ S%dE%d - %s\n", ep.SeasonNumber, ep.EpisodeNumber, ep.Title))
	}

	transaction := mailgun.NewMessage(Config.Mailgun.From, title, message.String(), recipients...)
	_, ID, err := Mailgun.Send(transaction)

	log.Printf("message id %s", ID)
	if err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}
}
