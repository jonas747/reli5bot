package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
)

type GeneralConfig struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	RefreshInterval int    `json:"refreshinterval"`
	Subreddit       string `json:"subreddit"`
	Comments        int    `json:"comments"`
	FlairTemplate   string `json:"flair_template_id"`
	FlairText       string `json:"flair_text"`
	Message         string `json:"message"`
	MessageSubject  string `json:"messagesubject"`
}

func LoadGeneralConfig() (*GeneralConfig, error) {
	b, err := ioutil.ReadFile("config.json")
	if err != nil {
		return nil, errors.New("config.json: " + err.Error())
	}
	var config GeneralConfig
	err = json.Unmarshal(b, &config)
	if err != nil {
		return nil, errors.New("config.json: " + err.Error())
	}
	return &config, nil
}

func LoadDataStorage() (*[]string, error) {
	b, err := ioutil.ReadFile("data.json")
	if err != nil {
		return nil, errors.New("data.json: " + err.Error())
	}

	var storage []string
	err = json.Unmarshal(b, &storage)
	if err != nil {
		return nil, errors.New("data.json: " + err.Error())
	}
	return &storage, nil
}

func SaveStorage(storage *[]string) error {
	marshalled, err := json.Marshal(storage)
	if err != nil {
		return err
	}

	ioutil.WriteFile("data.json", marshalled, os.ModePerm)
	return nil
}
