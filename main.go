package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"
	"time"
)

const DATA_PATH string = "site_data"

// TODO: check if the response time is too long of its an error(and if so write it and close thread)
/* TODO
https://fineproxy.org/ru/free-proxies/europe/russia/,
parse them
get all the best proxies
and then use the proxies to get  the results
*/

type Response struct {
	Time_mills int
	Bytes      int
	Code       int
	Date       time.Time
}

func request(wg *sync.WaitGroup, url string, ch chan Response) {
	defer wg.Done()

	start_time := time.Now()
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("ERROR: ", err)
	}

	defer resp.Body.Close()
	elapsed := time.Since(start_time)
	ch <- Response{
		Time_mills: int(elapsed),
		Code:       resp.StatusCode,
		Bytes:      int(resp.ContentLength),
		Date:       time.Now(),
	}
}

func process(iter int, url string) error {
	results := []Response{}

	var wg sync.WaitGroup

	ch := make(chan Response, iter*100)

	for i := 0; i < iter*100; i++ {
		wg.Add(1)
		go request(&wg, url, ch)
		time.Sleep(time.Millisecond * time.Duration((i*50 - (i * i))))
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for response := range ch {
		results = append(results, response)
	}

	if err := write_csv(&results, strconv.Itoa(iter)); err != nil {
		return err
	}

	return nil
}

func write_csv(reponses *[]Response, iter string) error {
	file_path := path.Join(DATA_PATH, "/", fmt.Sprintf("%s%s",iter, "data.csv"))
	file, err := os.Create(file_path)
	if err != nil {
		fmt.Println("ERROR: ", err)
		return err
	}

	defer file.Close()
	writer := csv.NewWriter(file)

	defer writer.Flush()

	// headers := []string{"date", "response time", "status code", "bytes"}
	// if err := writer.Write(headers); err != nil {
		// fmt.Println("ERROR WRITING TO FILE: ", err)
		// return err
	// }

	for _, v := range *reponses {
		record := []string{
			strconv.Itoa(int(v.Date.Unix())),
			strconv.Itoa(v.Time_mills),
			strconv.Itoa(v.Code),
			strconv.Itoa(v.Bytes),
		}
		if err := writer.Write(record); err != nil {
			fmt.Println("ERROR WRITING CSV TO FILE", err)
			return err
		}
	}
	return nil
}

func main() {
	err := os.Mkdir(DATA_PATH, 0666)
	if err != nil {
		panic(err)
	}

	url := flag.String("u", "", "URL which you are stress tasting")
	amount := flag.Int("a", 1, "Amount of hundreds of times to run")

	flag.Parse()

	for i := 1; i <= *amount; i++ {
		err := process(i, *url)
		if err != nil {
			fmt.Println("SOMETHING WENT SHIT", err)
		}
	}
}
