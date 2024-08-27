package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

const (
	dataPath = "data"
)

type MainData struct {
	Iteration int
	Data      []Response
	Url       string
}

type Response struct {
	TimeMillis        int
	Bytes             int
	Code              int
	Date              time.Time
	RequestsPerSecond float64
}

func process(iter int, url string) ([]Response, error) {
	results := []Response{}
	totalRequests := iter * 100

	// Calculate the width of the progress bar
	barWidth := 50
	progress := 0

	for i := 0; i < totalRequests; i++ {
		start := time.Now()

		// Make the request
		resp, err := http.Get(url)
		if err != nil {
			results = append(results, Response{
				TimeMillis: int(time.Since(start).Milliseconds()),
				Code:       0, // Error code
				Bytes:      0,
				Date:       time.Now(),
			})
			continue
		}
		defer resp.Body.Close()

		// Read the body to get the size
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			results = append(results, Response{
				TimeMillis: int(time.Since(start).Milliseconds()),
				Code:       resp.StatusCode,
				Bytes:      0,
				Date:       time.Now(),
			})
			continue
		}

		results = append(results, Response{
			TimeMillis: int(time.Since(start).Milliseconds()),
			Code:       resp.StatusCode,
			Bytes:      len(body),
			Date:       time.Now(),
		})

		// Update progress
		progress = i + 1
		percentage := float64(progress) / float64(totalRequests) * 100

		// Calculate filled portion
		filled := int(math.Round(float64(barWidth) * percentage / 100))
		bar := "[" + strings.Repeat("x", filled) + strings.Repeat(" ", barWidth-filled) + "]"

		// Print progress bar
		fmt.Printf("\rProgress: %s %.2f%% (%d/%d)", bar, percentage, progress, totalRequests)

		// Simulate some delay between requests to avoid overwhelming the server
		// time.Sleep(100 * time.Millisecond)
		time.Sleep(time.Millisecond * time.Duration((i*50 - (i * i))))
	}

	// Print a newline after the progress bar is complete
	fmt.Println()

	requestsPerSecond := float64(totalRequests) / float64(iter)

	// Add requests per second to each response
	for i := range results {
		results[i].RequestsPerSecond = requestsPerSecond
	}

	return results, nil
}

func main() {
	// url := "http://example.com"
	// amount := 3
	url := flag.String("u", "", "Website url to benchmark")
	amount := flag.Int("a", 3, "Iteration over requests (1 -> 100 requests, 2 -> 200 requests)")

	var global []MainData

	for i := 1; i <= *amount; i++ {
		newData := MainData{}

		data, err := process(i, *url)
		fmt.Println(data)
		if err != nil {
			fmt.Println("Error during processing:", err)
			continue
		}
		newData.Data = data
		newData.Iteration = i
		newData.Url = *url
		global = append(global, newData)
	}

	fmt.Println(global)
}
