package main

import (
	"fmt"
	"github.com/jonas747/reddit"
	"log"
	"time"
)

const (
	VERSION   = "1.0"
	USERAGENT = "RELI5 BOT version: " + VERSION + ". A bot that does stuff for /r/explainlikeimfive/ created by /u/jonas747"
)

func main() {
	log.Println("Starting RELI5 BOT version: " + VERSION + ". Loading config and logging in...")

	config, err := LoadGeneralConfig()
	if err != nil {
		log.Println(err)
		return
	}
	storage, err := LoadDataStorage()
	if err != nil {
		log.Println(err)
		return
	}

	account, err := reddit.Login(config.Username, config.Password, USERAGENT)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("Sucessfully logged in %s! Modhash: %s \n", account.Username, account.Modhash)

	Loop(config, *storage, account)
}

func Loop(config *GeneralConfig, storage []string, account *reddit.Account) {
	log.Println("comments: ", config.Comments, "; refreshinterval: ", config.RefreshInterval)

	//The reason config.refreshinterval is only used in comment stream is because you dont need to it inbox
	//stream since most likely you wont get 100 messages a minute
	cStream := &reddit.CommentStream{
		Update:        make(chan reddit.Comment),
		Stop:          make(chan bool),
		Errors:        make(chan error),
		FetchInterval: time.Duration(config.RefreshInterval) * time.Second,
		Subreddit:     config.Subreddit,
		RAccount:      account,
	}
	go cStream.Run()

	inboxStream := &reddit.InboxStream{
		Update:        make(chan reddit.PrivateMessage),
		Stop:          make(chan bool),
		Errors:        make(chan error),
		FetchInterval: time.Duration(60) * time.Second,
		Account:       account,
	}

	go inboxStream.Run()

	for {
		select {
		case comment := <-cStream.Update:
			rpost, err := reddit.GetPostFromId(comment.LinkId, USERAGENT)
			if err != nil {
				log.Println(err)
				continue
			}
			if rpost.Comments >= config.Comments && rpost.FlairText == "" {
				found := false
				for _, val := range storage {
					if val == comment.LinkId {
						found = true
						break
					}
				}
				if !found {
					//Message sent to author with 20+ comment threads
					link := fmt.Sprintf("http://www.reddit.com/message/compose/?to=%s&subject=flair_answered&message=%s", config.Username, rpost.FullName)
					modMLink := fmt.Sprintf("http://www.reddit.com/message/compose/?to=/r/%s", config.Subreddit)
					message := fmt.Sprintf(config.Message, rpost.Title, fmt.Sprintf("http://www.reddit.com/r/%s/comments/%s/", config.Subreddit, rpost.FullName[3:]), link, link, modMLink)
					account.Compose(config.MessageSubject, message, rpost.Author)
					log.Println("Sending message to ", rpost.Author, "; Titled: ", rpost.Title)
					storage = append(storage, comment.LinkId)
					SaveStorage(&storage)
				}
			}
		case cErr := <-cStream.Errors:
			log.Println(cErr)
		case message := <-inboxStream.Update:
			err := account.MarkMessageAsRead(message.FullName)
			if err != nil {
				log.Println(err)
				continue
			}
			body := message.Body
			subject := message.Subject
			if subject == "flair_answered" {
				p, err := reddit.GetPostFromId(body, USERAGENT)
				if err != nil {
					log.Println(err)
					continue
				}
				if p.Author != message.Author {
					log.Println("Tried flairing post with mismatched usernames!")
					continue
				}
				if p.FlairText != "" {
					log.Println("Post is already flaired, not flairing")
					continue
				}
				err = account.FlairPost(body, config.FlairTemplate, config.FlairText)
				if err != nil {
					log.Println(err)
					continue
				}
				log.Printf("Received message; Flairing post titled '%s' by '%s' \n", p.Title, p.Author)
			}
		case pmErr := <-inboxStream.Errors:
			log.Println(pmErr)
		}
	}
}
