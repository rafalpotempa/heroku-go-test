package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/go-github/github"
)

type webhook struct {
	Action     string
	Repository struct {
		ID       string
		FullName string
	}
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, []byte("my-secret-key"))
	if err != nil {
		log.Printf("error validating request body: err=%s\n", err)
		return
	}
	defer r.Body.Close()

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		log.Printf("could not parse webhook: err=%s\n", err)
		return
	}

	switch e := event.(type) {
	case *github.PushEvent:

	case *github.WatchEvent:
		// log.Printf("%v", e)
	case *github.StarEvent:
		// someone starred our repository
		if e.GetAction() == "created" {
			log.Printf("%s: %s starred repository %s\n", e.GetStarredAt(), e.GetSender(), e.GetRepo())
		} else if e.GetAction() == "delete" {
			log.Printf("%s: %s unstarred repository %s\n", e.GetStarredAt(), e.GetSender(), e.GetRepo())
		}
	default:
		log.Printf("unknown event type %s\n", github.WebHookType(r))
		return
	}
}

func index(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(w, "testy")
}

func main() {

	log.Println("server started")
	http.HandleFunc("/webhook", handleWebhook)
	http.HandleFunc("/", index)
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}
