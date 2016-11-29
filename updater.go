package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
)

const (
	BASE_URL = "//api.frankerfacez.com/v1/"
	SCHEME   = "http:"
	PER_PAGE = "200"
)

var wg sync.WaitGroup
var blacklist struct {
	Blacklist []string `json:"blacklist"`
}

type Emoticon struct {
	Id   uint64 `json:"id"`
	Name string `json:"name"`
}

type EmoticonsList struct {
	Links struct {
		Next string `json:"next"`
	} `json:"_links"`
	Pages         uint       `json:"_pages"`
	EmoticonsList []Emoticon `json:"emoticons"`
}

func (e *Emoticon) InBlacklist() bool {
	for _, s := range blacklist.Blacklist {
		if s == e.Name {
			return true
		}
	}
	return false
}

func update(output chan<- Emoticon) {
	var outputWg sync.WaitGroup

	resp, err := http.Get("https://raw.githubusercontent.com/Jiiks/BetterDiscordApp/master/data/emotefilter.json")
	if err != nil {
		log.Fatal("Failed fetching blacklist:", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &blacklist)
	// fmt.Println(blacklist)

	nextLink := SCHEME + BASE_URL + "emoticons?private=on&sort=updated-asc&per_page=" + PER_PAGE
	for {
		resp, err := http.Get(nextLink)
		if err != nil {
			log.Fatal("Failed fetching emotes:", err)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)

		var emoteList EmoticonsList
		json.Unmarshal(body, &emoteList)
		// fmt.Println(emoteList)

		outputWg.Add(1)
		go func(emotes []Emoticon) {
			defer outputWg.Done()

			for i := 0; i < len(emotes); i++ {
				if emotes[i].InBlacklist() {
					fmt.Println("Skipping blacklisted emote:", emotes[i].Name)
					continue
				}
				output <- emotes[i]
			}
		}(emoteList.EmoticonsList)

		if nextLink = emoteList.Links.Next; nextLink == "" {
			break
		}
	}
	outputWg.Wait()
	close(output)
}

func writeEmotesToFile(input <-chan Emoticon, filename string) {
	defer wg.Done()

	emotes := make(map[string]string)
	for s := range input {
		emotes[s.Name] = strconv.FormatUint(s.Id, 10)
		fmt.Printf("Indexed emotes: %d\r", len(emotes))
	}

	if jsonOutput, err := json.Marshal(emotes); err != nil {
		log.Fatal("Cannot marshal emotes to JSON:", err)
	} else {
		file, err := os.Create(filename)
		if err != nil {
			log.Fatal("Could not create file:", err)
		}
		defer file.Close()

		fmt.Println("Writing emotes to", filename)
		file.Write(jsonOutput)
	}
}

func main() {
	emoteChan := make(chan Emoticon, 200)
	filename := "emotes_ffz.json"

	if len(os.Args) > 1 {
		filename = os.Args[1]
	}

	wg.Add(1)
	go writeEmotesToFile(emoteChan, filename)
	update(emoteChan)
	wg.Wait()
}
