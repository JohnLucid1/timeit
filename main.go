package main

import (
	"flag"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Запускаю n колличесво горутин которые записывают скорость ответа сервера
// TODO: check if the response time is too long of its an error(and if so write it and close thread)
type Response struct {
	Time_mills uint
	Bytes      uint
	Code       uint
}

func process(url string, ch chan Response, wg *sync.WaitGroup) {
	defer wg.Done()

	url = fmt.Sprintf("%s%s", "https://", url)
	start := time.Now()

	r, err := http.Get(url)
	if err != nil {
		fmt.Println("ERROR: ", err)
		return
	}
	defer r.Body.Close()

	elapsed := time.Since(start)
	ch <- Response{Time_mills: uint(elapsed), Code: uint(r.StatusCode), Bytes: uint(r.ContentLength)}
}

// maybe write it so it'll go from 10 to 100 requests and measure how fast

func main() {
	url := flag.String("u", "", "URL which you are stress tasting")
	amount := flag.Int("a", 100, "Maximum amount of requests to send")
	flag.Parse()

	results := []Response{}
	var wg sync.WaitGroup
	ch := make(chan Response, *amount)

	for i := 0; i < *amount; i++ {
		wg.Add(1)
		go process(*url, ch, &wg)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for res := range ch {
		results = append(results, res)
	}

	fmt.Println(results)
}
