package utilities

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func ReadWords(fileLocation string) map[string]bool {
	file, err := os.Open("words.txt")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return nil
	}
	defer file.Close()

	// Create a map to store words
	wordMap := make(map[string]bool)

	// Use a scanner to read the file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Split the line into words
		words := strings.Fields(line)
		for _, word := range words {
			// Update the map with word count
			wordMap[word] = true
		}
	}

	// Check for errors in the scanning process
	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
		return nil
	}

	return wordMap
}
