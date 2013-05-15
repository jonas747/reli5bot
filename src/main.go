package main

import (
	"fmt"
	simplejson "github.com/bitly/go-simplejson"
	"github.com/jonas747/reddit"
	"net/url"
	"time"
)

const (
	VERSION   = "0.1b"
	USERAGENT = "RELI5 BOT version: " + VERSION + ". A bot that does stuff for /r/explainlikeimfive/ created by /u/jonas747"
)

func main() {
	fmt.Println("Starting RELI5 BOT version: " + VERSION + ". Loading config and logging in...")

	config, err := LoadGeneralConfig()
	if err != nil {
		fmt.Println(err)
		return
	}
	storage, err := LoadDataStorage()
	if err != nil {
		fmt.Println(err)
		return
	}

	account, err := reddit.Login(config.Username, config.Password, USERAGENT)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Sucessfully logged in %s! Modhash: %s \n", account.Username, account.Modhash)

	Loop(config, *storage, account)
}

func Loop(config *GeneralConfig, storage []string, account *reddit.Account) {
	after := ""
	afterTime := 0
	ticker := time.NewTicker(time.Duration(config.RefreshInterval) * time.Second)
	fmt.Println("comments: ", config.Comments, "; refreshinterval: ", config.RefreshInterval)
	for {
		<-ticker.C

		locafter := after
		locafterTime := afterTime

		//fmt.Println("Ticked!")
		///////////////////////////
		// Gets the recent comments, checks parent and send a message is neceseraarry
		//////////////////////////
		var json simplejson.Json
		var err error
		if after != "" {
			json, err = reddit.Get("http://www.reddit.com/r/"+config.Subreddit+"/comments.json", USERAGENT, url.Values{"count": {"100"}}, nil)

		} else {
			json, err = reddit.Get("http://www.reddit.com/r/"+config.Subreddit+"/comments.json", USERAGENT, url.Values{"count": {"100"}}, nil)
		}
		if err != nil {
			fmt.Println(err)
			return
		}
		posts, err := reddit.RCommentsFromListingJson(json)
		if err != nil {
			fmt.Println(err)
			return
		}
		for _, comment := range posts {
			if comment.Created <= afterTime {
				continue
			}
			rpost, err := reddit.GetPostFromId(comment.LinkId, USERAGENT)
			if err != nil {
				fmt.Println(err)
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
					account.Compose("Have your ELI5 post been answered?", message, rpost.Author)
					fmt.Println("Sending message to ", rpost.Author, "; Titled: ", rpost.Title)
					storage = append(storage, comment.LinkId)
					SaveStorage(&storage)
				}
			}
			if comment.Created > locafterTime {
				locafter = comment.FullName
				locafterTime = comment.Created
			}

		}

		afterTime = locafterTime
		after = locafter
		///////////////////////
		// Checks inbox and flairs if command is sent
		//////////////////////
		inbox, err := account.GetInbox(true)
		if err != nil {
			fmt.Println(err)
			continue
		}
		for _, message := range inbox {
			err := account.MarkMessageAsRead(message)
			if err != nil {
				fmt.Println(err)
				continue
			}
			body := message.Body
			subject := message.Subject
			if subject == "flair_answered" {
				p, err := reddit.GetPostFromId(body, USERAGENT)
				if err != nil {
					fmt.Println(err)
					continue
				}
				if p.Author != message.Author {
					fmt.Println("Tried flairing post with mismatched usernames!")
					continue
				}
				if p.FlairText != "" {
					fmt.Println("Post is already flaired, not flairing")
					continue
				}
				err = account.FlairPost(body, config.FlairTemplate, config.FlairText)
				if err != nil {
					fmt.Println(err)
					continue
				}
				fmt.Printf("Received message; Flairing post titled '%s' by '%s' \n", p.Title, p.Author)
			}
		}
	}
}
