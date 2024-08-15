package application

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/victowork/firefly/utilities"
)

// Configuration for rate limiting,  as per this configuration after a initial set of 5000 requests service going to rest for 1 millisecond before processing the next.
const (
	rateLimitInterval     = 1 * time.Millisecond
	maxConcurrentRequests = 5000
)

// maps to store word bank and the web essay urls
var wordBank = make(map[string]bool)
var essayLinks = make(map[string]bool)

type EssayWordCounter struct {
}

type IessayWordCounter interface {
	CountEssayWords()
	fetchWebEssayContent(url string, results chan<- string, rateLimiter <-chan time.Time, wg *sync.WaitGroup)
	processEssayContent(content string, wordCount map[string]int)
}

// use regex for finding proper words from the essay & use word bank map to cross check the valid words
func (e *EssayWordCounter) processEssayContent(content string, wordCount map[string]int64) {
	wordRegexp := regexp.MustCompile(`\b[a-zA-Z]{3,}\b`)
	words := wordRegexp.FindAllString(strings.ToLower(content), -1)

	for _, word := range words {
		if _, ok := wordBank[word]; ok {
			wordCount[word]++
		}
	}
}

func (e *EssayWordCounter) fetchWebEssayContent(url string, results chan<- string, rateLimiter <-chan time.Time, wg *sync.WaitGroup) {

	defer wg.Done()

	fmt.Println("started getting content from: ", url, "on: ", time.Now())
	<-rateLimiter // Rate limit control

	client := http.Client{ // setting web client timeout to 2 seconds
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(url) // getting the web content
	if err != nil {
		fmt.Println("Error fetching"+url+":", err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: "+url+" returned status: ", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading body from "+url+":", err)
		return
	}

	results <- string(body)

}

func (e *EssayWordCounter) CountEssayWords() {
	//Read wordbank and web urls from local files
	essayLinks = utilities.ReadDataLocal("./resources/endg-urls.txt", true)
	wordBank = utilities.ReadDataLocal("./resources/wordBank.txt", false)

	var wg sync.WaitGroup
	results := make(chan string, len(essayLinks))
	rateLimiter := time.NewTicker(rateLimitInterval).C

	// Create a semaphore to limit concurrent requests
	sem := make(chan struct{}, maxConcurrentRequests)

	// Fetch content from each URL
	now := time.Now()
	fmt.Println("web get started on: ", now)
	for url := range essayLinks {
		wg.Add(1)
		go func(url string) {
			defer func() { <-sem }()
			sem <- struct{}{}
			e.fetchWebEssayContent(url, results, rateLimiter, &wg)
		}(url)
	}

	// Wait for all fetches to complete
	wg.Wait()
	close(results)

	wordCount := make(map[string]int64)

	// Process results
	for content := range results {
		e.processEssayContent(content, wordCount)
	}

	// Convert word counts to slice and sort for result
	type wordOutput struct {
		Word  string `json:"word"`
		Count int64  `json:"count"`
	}

	// create slice from the map
	var sortedwordOutput []wordOutput
	for word, count := range wordCount {
		sortedwordOutput = append(sortedwordOutput, wordOutput{Word: word, Count: count})
	}

	// sort the word DESC on count
	sort.Slice(sortedwordOutput, func(i, j int) bool {
		if sortedwordOutput[i].Count == sortedwordOutput[j].Count {
			return sortedwordOutput[i].Word < sortedwordOutput[j].Word
		}
		return sortedwordOutput[i].Count > sortedwordOutput[j].Count
	})

	// Limit to top 10 words
	if len(sortedwordOutput) > 10 {
		sortedwordOutput = sortedwordOutput[:10]
	}

	// Marshal to JSON for result set
	jsonOutput, err := json.MarshalIndent(sortedwordOutput, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	// Print the result & log total time
	fmt.Println(string(jsonOutput))
	fmt.Println("time taken for total process: ", time.Since(now))
}
