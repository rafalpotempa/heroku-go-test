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
		log.Printf("error validating request body: err=\"%s\"\n", err)
		return
	}
	defer r.Body.Close()

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		log.Printf("could not parse webhook: err=\"%s\"\n", err)
		return
	}

	switch e := event.(type) {
	case *github.PushEvent:
		log.Printf("push")
	case *github.WatchEvent:
		log.Printf("%s is watching repo \"%s\"\n", e.GetSender().GetLogin(), e.GetRepo().GetFullName())
	case *github.StarEvent:
		// someone starred our repository
		if e.GetAction() == "created" {
			log.Printf("repository starred\n")
		} else if e.GetAction() == "deleted" {
			log.Printf("repository unstarred\n")
		}
	default:
		log.Printf("unknown event type: \"%s\"\n", github.WebHookType(r))
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
