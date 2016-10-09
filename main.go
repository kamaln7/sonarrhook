package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/kamaln7/sonarrhook/config"

	"gopkg.in/mailgun/mailgun-go.v1"
)

var (
	c  config.Obj
	mg mailgun.Mailgun
)

func main() {
	log.Print("reading config file")
	c = config.Read()

	http.HandleFunc("/download", download)

	mg = mailgun.NewMailgun(c.Mailgun.Domain, c.Mailgun.APIKey, c.Mailgun.PublicAPIKey)

	listenAddr := fmt.Sprintf("%s:%d", c.HTTP.Host, c.HTTP.Port)
	log.Printf("listening on %s\n", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

// A DownloadEvent is an event sent by Sonarr that includes
// info about the new episodes
type DownloadEvent struct {
	EventType string
	Series    Series
	Episodes  []Episode
}

// A Series is a sonarr series
type Series struct {
	ID          int
	Title, Path string
	TvdbID      uint64
}

// An Episode is a file that was downloaded by Sonarr
type Episode struct {
	ID                                                           int
	EpisodeNumber, SeasonNumber, QualityVersion                  int
	Title, AirDate, AirDateUtc, Quality, ReleaseGroup, SceneName string
}

func download(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)

		return
	}

	if apiKey := r.URL.Query().Get("key"); apiKey != c.HTTP.Key {
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

	log.Printf("got series (id %d) [%s]", event.Series.ID, event.Series.Title)

	var contactNames, recipients []string

	contactNames, ok := c.Series[event.Series.Title]
	if !ok {
		log.Printf("series (id %d) %s not in config", event.Series.ID, event.Series.Title)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)

		return
	}

	for _, name := range contactNames {
		if _, ok := c.Contacts[name]; !ok {
			log.Printf("contact %s not in config", name)

			continue
		}

		recipients = append(recipients, c.Contacts[name])
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

	transaction := mailgun.NewMessage(c.Mailgun.From, title, message.String(), recipients...)
	_, ID, err := mg.Send(transaction)

	log.Printf("message id %s", ID)
	if err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}
}
