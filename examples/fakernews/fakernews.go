package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/itsByte/gomarkov"
)

const (
	hnBaseURL        = "https://hacker-news.firebaseio.com/v0/"
	hnTopStoriesPath = "topstories.json"
	hnStoryItemPath  = "item/"
)

type hnStory struct {
	Title string `json:"title"`
}

func main() {
	train := flag.Bool("train", false, "Train the markov chain")
	flag.Parse()
	if *train {
		chain, err := buildModel()
		if err != nil {
			fmt.Println(err)
			return
		}
		defer chain.Close()
	} else {
		chain, err := loadChain()
		if err != nil {
			fmt.Println(err)
			return
		}
		defer chain.Close()
		generateHNStory(chain)
	}
}

func loadChain() (*gomarkov.Chain, error) {
	storage, err := gomarkov.NewPebbleStorage("db")
	if err != nil {
		return nil, err
	}
	return gomarkov.NewChain(3, storage), nil
}

func buildModel() (*gomarkov.Chain, error) {
	stories, err := fetchHNTopStories()
	if err != nil {
		return nil, err
	}
	chain, err := loadChain()
	if err != nil {
		return nil, err
	}
	var wg sync.WaitGroup
	wg.Add(len(stories))
	fmt.Println("Adding HN story titles to markov chain...")
	for _, storyID := range stories {
		go func(storyID int) {
			defer wg.Done()
			story, err := fetchHNStory(storyID)
			if err != nil {
				fmt.Println(err)
				return
			}
			chain.Add(1, strings.Split(story.Title, " "))
		}(storyID)
	}
	wg.Wait()
	return chain, nil
}

func fetchHNTopStories() ([]int, error) {
	fmt.Println("Fetching HN top stories...")
	resp, err := http.Get(fmt.Sprintf("%s%s", hnBaseURL, hnTopStoriesPath))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var stories []int
	err = json.Unmarshal(body, &stories)
	return stories, err
}

func fetchHNStory(storyID int) (hnStory, error) {
	var story hnStory
	resp, err := http.Get(fmt.Sprintf("%s%s%d.json", hnBaseURL, hnStoryItemPath, storyID))
	if err != nil {
		return story, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return story, err
	}
	err = json.Unmarshal(body, &story)
	return story, err
}

func generateHNStory(chain *gomarkov.Chain) {
	res, err := chain.GenerateAll(1)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(strings.Join(res, " "))
}
