package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

const (
	PORTION = 100
)

type Data struct {
	Iteration int
	Data      []Response
	URL       string
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

	// Initial sleep time in milliseconds (e.g., 1000ms = 1 second)
	initialSleep := 1000.0

	// Track the start time of the entire process to calculate requests per second
	startTime := time.Now()

	for i := range totalRequests {
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

		// Read the body to get the size
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close() // Close the response body after reading
		if err != nil {
			results = append(results, Response{
				TimeMillis: int(time.Since(start).Milliseconds()),
				Code:       resp.StatusCode,
				Bytes:      0,
				Date:       time.Now(),
			})

			continue
		}

		// Track the time taken and store the response
		elapsed := time.Since(start).Milliseconds()
		results = append(results, Response{
			TimeMillis: int(elapsed),
			Code:       resp.StatusCode,
			Bytes:      len(body),
			Date:       time.Now(),
		})

		// Update progress
		progress = i + 1
		percentage := float64(progress) / float64(totalRequests) * 100

		// Calculate filled portion
		filled := int(math.Round(float64(barWidth) * percentage / PORTION))
		bar := "[" + strings.Repeat("x", filled) + strings.Repeat(" ", barWidth-filled) + "]"

		// Print progress bar
		fmt.Printf("\rProgress: %s %.2f%% (%d/%d)", bar, percentage, progress, totalRequests)

		// Reduce sleep duration progressively
		sleepTime := initialSleep / math.Sqrt(float64(progress))
		if sleepTime < 1 {
			sleepTime = 1 // Ensure there's at least a 1 millisecond delay
		}
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	}

	// Print a newline after the progress bar is complete
	fmt.Println()

	// Calculate the total elapsed time
	totalElapsedTime := time.Since(startTime).Seconds()

	// Calculate requests per second for the entire operation
	requestsPerSecond := float64(totalRequests) / totalElapsedTime

	// Add requests per second to each response
	for i := range results {
		results[i].RequestsPerSecond = requestsPerSecond
	}

	return results, nil
}

func createPlot(global []Data) {
	if err := termui.Init(); err != nil {
		fmt.Println("failed to initialize termui:", err)

		return
	}
	defer termui.Close()

	// Prepare data for the plot, sorting responses by Date
	allResponses := make([]Response, 0)
	for _, data := range global {
		allResponses = append(allResponses, data.Data...)
	}

	sort.Slice(allResponses, func(i, j int) bool {
		return allResponses[i].Date.Before(allResponses[j].Date)
	})

	// Prepare data for the plot
	var dataSeries []float64
	var labels []string
	var colors []termui.Color
	var totalResponseTime int
	var totalRequests int
	non200Requests := 0
	non200RequestsPerSecond := 0.0

	for _, data := range global {
		for _, response := range data.Data {
			dataSeries = append(dataSeries, float64(response.TimeMillis))
			labels = append(labels, strconv.Itoa(response.Code))
			colors = append(colors, termui.ColorWhite) // Always white for response time

			// Accumulate total response time and count requests
			totalResponseTime += response.TimeMillis
			totalRequests++

			if response.Code != http.StatusOK {
				non200Requests++
				non200RequestsPerSecond += response.RequestsPerSecond / float64(len(global))
			}
		}
	}

	// Calculate average response time
	averageResponseTime := float64(totalResponseTime) / float64(totalRequests)

	// Create a bar chart
	barChart := widgets.NewBarChart()
	barChart.Data = dataSeries
	barChart.Labels = labels
	barChart.BarColors = colors
	barChart.LabelStyles = make([]termui.Style, len(labels))

	// Set all label styles to white
	for i := range barChart.LabelStyles {
		barChart.LabelStyles[i] = termui.NewStyle(termui.ColorWhite)
	}

	barChart.Title = "Response Time vs Number of Requests"
	barChart.SetRect(0, 0, 100, 20)

	// Prepare the summary to be displayed
	summary := widgets.NewParagraph()
	summary.Text = fmt.Sprintf("Average Response Time: %.2f ms\nNon-200 Status Codes: %d requests\nRequests per Second for Non-200 Codes: %.2f", averageResponseTime, non200Requests, non200RequestsPerSecond)
	summary.SetRect(0, 20, 100, 25)

	termui.Render(barChart, summary)

	uiEvents := termui.PollEvents()
	for {
		e := <-uiEvents
		if e.Type == termui.KeyboardEvent {
			break
		}
	}
}

func main() {
	url := flag.String("u", "", "Website url to benchmark")
	amount := flag.Int("a", 3, "Iteration over requests (1 -> 100 requests, 2 -> 200 requests)")
	is_multithreaded := flag.Bool("m", false, "Instead of sending requests one after another, send all at once")

	flag.Parse()
	if !*is_multithreaded {
		var global []Data

		for i := 1; i <= *amount; i++ {

			data, err := process(i, *url)
			if err != nil {
				fmt.Println("Error during processing:", err)

				continue
			}

			global = append(global, Data{
				Data:      data,
				Iteration: i,
				URL:       *url,
			})
		}

		createPlot(global)
	} else {
		maxRequests := measureMultithreadedRequests(*url)
		fmt.Printf("MAximum concurrent requests before errors: %d\n", maxRequests)
	}
}

func measureMultithreadedRequests(url string) int {
	maxRequests := 0
	increment := 10
	for requests := increment; ; requests += increment {
		results, err := sendMultithreadedRequests(requests, url)
		if err != nil {
			fmt.Printf("Error sending %d requests: %v\n", requests, err)
			break
		}

		// Check if any response has a non-200 status code
		hasNon200 := false
		for _, r := range results {
			if r.Code != http.StatusOK {
				hasNon200 = true
				break
			}
		}

		if hasNon200 {
			break
		}

		maxRequests = requests
	}
	return maxRequests
}

func sendMultithreadedRequests(numRequests int, url string) ([]Response, error) {
	results := make([]Response, numRequests)
	var wg sync.WaitGroup
	wg.Add(numRequests)

	for i := 0; i < numRequests; i++ {
		go func(index int) {
			defer wg.Done()
			start := time.Now()
			resp, err := http.Get(url)
			if err != nil {
				results[index] = Response{
					TimeMillis: int(time.Since(start).Milliseconds()),
					Code:       0, // Error code
					Bytes:      0,
					Date:       time.Now(),
				}
				return
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				results[index] = Response{
					TimeMillis: int(time.Since(start).Milliseconds()),
					Code:       resp.StatusCode,
					Bytes:      0,
					Date:       time.Now(),
				}
				return
			}
			results[index] = Response{
				TimeMillis: int(time.Since(start).Milliseconds()),
				Code:       resp.StatusCode,
				Bytes:      len(body),
				Date:       time.Now(),
			}
		}(i)
	}

	wg.Wait()
	return results, nil
}
