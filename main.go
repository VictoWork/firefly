package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"github.com/victowork/chi-api/utilities"
	
)

// Configuration for rate limiting
const (
	rateLimitInterval     = 100 * time.Millisecond
	maxConcurrentRequests = 5
)

// Word bank

wordBank:=ReadWords()


func fetchContent(url string, results chan<- string, rateLimiter <-chan time.Time, wg *sync.WaitGroup) {
	defer wg.Done()

	<-rateLimiter // Rate limit control

	resp, err := http.Get(url)
	if err != nil {
		results <- fmt.Sprintf("Error fetching %s: %v", url, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		results <- fmt.Sprintf("Error: %s returned status %d", url, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		results <- fmt.Sprintf("Error reading body from %s: %v", url, err)
		return
	}

	results <- string(body)
}

func processContent(content string, wordCount map[string]int) {
	wordRegexp := regexp.MustCompile(`\b[a-zA-Z]{3,}\b`)
	words := wordRegexp.FindAllString(strings.ToLower(content), -1)

	for _, word := range words {
		if _, ok := wordBank[word]; ok {
			wordCount[word]++
		}
	}
}

func main() {
	// Example URLs (replace with actual URLs)
	essays := []string{
		"https://www.engadget.com/2019/08/25/sony-and-yamaha-sc-1-sociable-cart/",
		// "https://www.engadget.com/2019/08/24/trump-tries-to-overturn-ruling-stopping-him-from-blocking-twitte/",
		// "https://www.engadget.com/2019/08/24/crime-allegation-in-space/",
	}

	var wg sync.WaitGroup
	results := make(chan string, len(essays))
	rateLimiter := time.NewTicker(rateLimitInterval).C

	// Create a semaphore to limit concurrent requests
	sem := make(chan struct{}, maxConcurrentRequests)

	// Fetch content from each URL
	for _, url := range essays {
		wg.Add(1)
		func(url string) {
			defer func() { <-sem }()
			sem <- struct{}{}
			fetchContent(url, results, rateLimiter, &wg)
		}(url)
	}

	// Wait for all fetches to complete
	wg.Wait()
	close(results)

	wordCount := make(map[string]int)

	// Process results
	for content := range results {
		processContent(content, wordCount)
	}

	// Convert word counts to slice and sort
	type wordFrequency struct {
		Word  string `json:"word"`
		Count int    `json:"count"`
	}

	var sortedWordFreqs []wordFrequency
	for word, count := range wordCount {
		sortedWordFreqs = append(sortedWordFreqs, wordFrequency{Word: word, Count: count})
	}

	sort.Slice(sortedWordFreqs, func(i, j int) bool {
		if sortedWordFreqs[i].Count == sortedWordFreqs[j].Count {
			return sortedWordFreqs[i].Word < sortedWordFreqs[j].Word
		}
		return sortedWordFreqs[i].Count > sortedWordFreqs[j].Count
	})

	// Limit to top 10 words
	if len(sortedWordFreqs) > 10 {
		sortedWordFreqs = sortedWordFreqs[:10]
	}

	// Marshal to JSON
	jsonOutput, err := json.MarshalIndent(sortedWordFreqs, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	// Print the result
	fmt.Println(string(jsonOutput))
}
