package main

import (
	//"fmt"
	"bytes"
	"github.com/jonas747/reddit"
	"log"
	"text/template"
	"time"
)

const (
	VERSION   = "1.2"
	USERAGENT = "RELI5 BOT version: " + VERSION + ". A bot that does stuff for /r/explainlikeimfive/ created by /u/jonas747"
)

var messageTemplate *template.Template

type templateStruct struct {
	Post   reddit.Thing
	Config GeneralConfig
}

func main() {
	log.Println("Starting RELI5 BOT version: " + VERSION + ". Loading config and datastore then logging in...")
	config, err := LoadGeneralConfig()
	if err != nil {
		log.Println(err)
		return
	}

	messageTemplate, err = template.New("messagebody").Parse(config.Message)
	if err != nil {
		log.Println("Error parsing message template: ", err)
	}

	log.Println("Starting main loop...")
	for {
		timer := time.After(time.Duration(2) * time.Second)

		storage, err := LoadDataStorage()
		if err != nil {
			log.Println(err)
			return
		}

		Loop(config, *storage)
		<-timer
	}
}

func Loop(config *GeneralConfig, storage []string) {

	account, err := reddit.Login(config.Username, config.Password, USERAGENT)
	if err != nil {
		log.Println(err)
		return
	}

	//The reason config.refreshinterval is only used in comment stream is because you dont need to it inbox
	//stream since most likely you wont get 100 messages a minute
	cStream := &reddit.CommentStream{
		Update:        make(chan reddit.Thing),
		Stop:          make(chan bool),
		Errors:        make(chan error),
		FetchInterval: time.Duration(config.RefreshInterval) * time.Second,
		Subreddit:     config.Subreddit,
		RAccount:      account,
	}
	go cStream.Run()

	inboxStream := &reddit.InboxStream{
		Update:        make(chan reddit.Thing),
		Stop:          make(chan bool),
		Errors:        make(chan error),
		FetchInterval: time.Duration(5) * time.Second,
		Account:       account,
	}

	go inboxStream.Run()

MainLoop:
	for {
		select {
		case comment := <-cStream.Update:
			for _, val := range storage {
				if val == comment.Data.Link_id {
					continue MainLoop
				}
			}

			response, err := reddit.GetPostFromId(comment.Data.Link_id, USERAGENT)
			if err != nil && len(response.Data.Children) < 1 {
				log.Println(err)
				continue
			}

			if len(response.Data.Children) < 1 {
				log.Println("Cannot find parent post to comment id ", comment.Data.Link_id)
				continue
			}
			// Ugly, i know
			post := response.Data.Children[0]

			if post.Data.Num_comments >= config.Comments && post.Data.Link_flair_text == "" {

				tmplStruct := templateStruct{
					post,
					*config,
				}

				buffer := new(bytes.Buffer)
				err = messageTemplate.Execute(buffer, tmplStruct)
				if err != nil {
					log.Println("Error executing template: ", err)
					continue MainLoop
				}
				message := string(buffer.Bytes())
				account.Compose(config.MessageSubject, message, post.Data.Author)
				log.Printf("Sending message to %s, The post title: %s\n", post.Data.Author, post.Data.Title)
				storage = append(storage, comment.Data.Link_id)
				SaveStorage(&storage)
			}
		case cErr := <-cStream.Errors:
			log.Println(cErr)
			if cErr == reddit.ERRCOMMENTSVOID {
				log.Println("Restarting")
				return
			}
		case message := <-inboxStream.Update:
			err := account.MarkMessageAsRead(message.Data.Name)
			if err != nil {
				log.Println(err)
				continue
			}
			body := message.Data.Body
			subject := message.Data.Subject
			if subject == "flair_answered" || subject == "flair answered" || subject == "flair" {
				response, err := reddit.GetPostFromId(body, USERAGENT)
				if err != nil && len(response.Data.Children) < 1 {
					log.Println(err)
					continue
				}
				if len(response.Data.Children) < 1 {
					log.Println("Cannot find post with id ", body)
					continue
				}
				post := response.Data.Children[0]
				if post.Data.Author != message.Data.Author {
					log.Println("Tried flairing post with mismatched usernames!")
					continue
				}
				if post.Data.Link_flair_text != "" {
					log.Println("Post is already flaired, not flairing")
					continue
				}

				err = account.FlairPost(body, config.FlairTemplate, config.FlairText)
				if err != nil {
					log.Println(err)
					continue
				}
				log.Printf("Received message, Flairing post titled '%s' by '%s'\n", post.Data.Title, post.Data.Author)
			}
		case pmErr := <-inboxStream.Errors:
			log.Println(pmErr)
		}
	}
}
