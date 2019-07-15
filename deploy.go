// /*
// HookHandler - listen for github webhooks, sending updates on channel.
// DeploymentMonitor select update type based on channel and call deployment script
// */
// package main

// import (
// 	"fmt"
// 	"html/template"
// 	"io/ioutil"
// 	"log"
// 	"net/http"

// 	"github.com/google/go-github/github"
// 	"github.com/nlopes/slack"
// )

// // HookHandler parses GitHub webhooks and sends an update to DeploymentMonitor.
// func HookHandler(prUp chan<- PullUpdate, cUp chan<- CommitUpdate, brUp chan<- BranchUpdate) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		payload, err := ioutil.ReadAll(r.Body)
// 		if err != nil {
// 			log.Printf("error reading request body: err=%s\n", err)
// 			return
// 		}
// 		defer r.Body.Close()
// 		event, err := github.ParseWebHook(github.WebHookType(r), payload)
// 		if err != nil {
// 			log.Printf("could not parse webhook: err=%s\n", err)
// 			return
// 		}
// 		// send PR or Branch updates to the DeploymentMonitor
// 		// send commit status (from CircleCI) to DeploymentMonitor
// 		switch e := event.(type) {
// 		case *github.StatusEvent:
// 			var commitMessage string
// 			if e.Commit != nil {
// 				if e.Commit.Commit != nil {
// 					commitMessage = *e.Commit.Commit.Message
// 				}
// 			}
// 			cUp <- CommitUpdate{status: *e.State, sha: *e.SHA, message: commitMessage}
// 			return
// 		case *github.PullRequestEvent:
// 			prUp <- PullUpdate{
// 				pr: PullRequest{
// 					Number: *e.Number,
// 					SHA:    *e.PullRequest.Head.SHA,
// 				},
// 				action: *e.Action,
// 			}
// 			return
// 		case *github.PushEvent:
// 			ref := *e.Ref
// 			branch := ref[len("refs/heads/"):]
// 			if branch == "master" {
// 				brUp <- BranchUpdate{
// 					Name: branch,
// 					SHA:  *e.After,
// 				}
// 			}
// 			return
// 		default:
// 			log.Printf("unknown WebHookType: %s, webhook-id: %s skipping\n", github.WebHookType(r), r.Header.Get("X-GitHub-Delivery"))
// 			return
// 		}
// 	}
// }

// // DeploymentMonitor receives updates when
// // a pull request is opened/updated/closed
// // a branch receives a new push(merge to master is a push event)
// // the pullUpdate and branchUpdate channels will update a branch or PR SHA
// // to the current one.
// // Later, a commit status will come through. Deploment Monitor will find which branch
// // The commit belongs to, and deploy that pull request.
// func DeploymentMonitor(dm deployer, botUpdates chan<- botEvent) (chan<- PullUpdate, chan<- CommitUpdate, chan<- BranchUpdate) {
// 	prUp := make(chan PullUpdate)
// 	cUp := make(chan CommitUpdate)
// 	brUp := make(chan BranchUpdate)

// 	pulls := make(map[int]PullRequest)
// 	// map[branchName]commitSHA
// 	branches := make(map[string]string)
// 	// map[commitSHA]status
// 	// tracking commits to avoid duplicates
// 	commits := make(map[string]string)

// 	// load templates
// 	tmpl, err := template.ParseFiles(
// 		"templates/branch-deployment.template",
// 		"templates/branch-service.template",
// 		"templates/pr-deployment.template",
// 		"templates/pr-service.template",
// 	)
// 	if err != nil {
// 		log.Fatalf("failed to parse template files, err: %s\n", err)
// 	}
// 	go func() {
// 		for {
// 			select {
// 			case p := <-prUp:
// 				pulls[p.pr.Number] = p.pr
// 				commits[p.pr.SHA] = "pending"
// 				log.Printf("updated pr: %d to commit: %s, action=%s\n", p.pr.Number, p.pr.SHA, p.action)
// 				// TODO: if action = closed, teardown a deployment?
// 			case br := <-brUp:
// 				branches[br.Name] = br.SHA
// 				commits[br.SHA] = "pending"
// 				log.Printf("updated branch: %s to commit: %s", br.Name, br.SHA)
// 			case c := <-cUp:
// 				// check pull requests
// 				for _, d := range pulls {
// 					if d.SHA == c.sha && c.status == "success" {
// 						if status, ok := commits[c.sha]; ok && status == "deployed" {
// 							continue
// 						}
// 						fmt.Printf("deploying pr=%d, commit=%s\n", d.Number, d.SHA)
// 						if err := dm.deployPR(d.Number, d.SHA, tmpl); err != nil {
// 							log.Println(err)
// 							break
// 						}
// 						botUpdates <- botEvent{
// 							attachments: []slack.Attachment{slack.Attachment{
// 								Fallback:   "Pull Request Deployment",
// 								Color:      "#36a64f",
// 								AuthorName: "Github Discussion",
// 								AuthorLink: fmt.Sprintf("https://github.com/acme/acme-ose/pull/%d", d.Number),
// 								Title:      fmt.Sprintf("%d.pr.acme.net", d.Number),
// 								TitleLink:  fmt.Sprintf("https://%d.pr.acme.net", d.Number),
// 								Pretext:    fmt.Sprintf("Deployed PR %d, commit %s", d.Number, d.SHA),
// 								Text:       c.message,
// 							}},
// 						}
// 						commits[c.sha] = "deployed"
// 						break
// 					}
// 				}
// 				// check branches
// 				for branch, sha := range branches {
// 					if sha == c.sha && c.status == "success" {
// 						// check if commit already marked as deployed
// 						// to avoid duplicates
// 						if status, ok := commits[sha]; ok && status == "deployed" {
// 							continue
// 						}
// 						fmt.Printf("deploying branch=%s, commit=%s\n", branch, sha)
// 						if err := dm.deployBranch(branch, sha, tmpl); err != nil {
// 							log.Println(err)
// 							break
// 						}
// 						// send update to slack
// 						botUpdates <- botEvent{
// 							attachments: []slack.Attachment{slack.Attachment{
// 								Fallback:  "Branch Deployment",
// 								Color:     "#36a64f",
// 								Title:     "acme.acme.net",
// 								TitleLink: "https://acme.acme.net",
// 								Pretext:   fmt.Sprintf("Deployed Branch %s, commit %s", branch, sha),
// 								Text:      c.message,
// 							}},
// 						}
// 						commits[sha] = "deployed"
// 						break
// 					}
// 				}
// 			}
// 		}
// 	}()
// 	return prUp, cUp, brUp
// }
