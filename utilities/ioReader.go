package utilities

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// utility method for reading resources (wordbank & list of url from constant files)
func ReadDataLocal(fileLocation string, isessay bool) map[string]bool {
	file, err := os.Open(fileLocation)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return nil
	}
	defer file.Close()

	// Create a map to store data
	dataMap := make(map[string]bool)

	// Use a scanner to read the file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if isessay {
			dataMap[line] = true
		} else {
			// Split the line into words
			words := strings.Fields(line)
			for _, word := range words {
				// Update the map with word count
				dataMap[word] = true
			}
		}

	}

	// Check for errors in the scanning process
	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
		return nil
	}

	return dataMap
}
