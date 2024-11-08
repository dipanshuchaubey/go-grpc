package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// func generator(limit int, ch chan<- int) {
// 	for i := 2; i < limit; i++ {
// 		ch <- i
// 	}

// 	close(ch)
// }

// func filter(ch <-chan int, filterCh chan<- int, num int) {

// }

// func main() {
// 	nums := make(chan int)

// 	go generator(100, nums)

// 	for {
// 		filter(nums, )
// 	}
// }

type result struct {
	url     string
	err     error
	latency time.Duration
}

func benchmark(url string, ch chan<- result) {
	start := time.Now()

	res, httpErr := http.Get(url)
	if httpErr != nil {
		ch <- result{url, httpErr, time.Since(start)}
	} else {
		ch <- result{url, nil, time.Since(start).Round(time.Millisecond)}
		res.Body.Close()
	}
}

func main() {
	start := time.Now()

	links := []string{
		"https://google.com",
		"https://facebook.com",
		"https://x.com",
		"https://dipanshu.work",
	}

	results := make(chan result)

	for _, link := range links {
		go benchmark(link, results)
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		for range links {
			res := <-results
			if res.err != nil {
				fmt.Printf("Error getting stats for %v: %v", res.url, res.err)
			} else {
				fmt.Printf("Called %v. Took: %v", res.url, res.latency)
				fmt.Println()
			}
		}
	}()

	wg.Add(1)
	go func(ch <-chan result) {
		defer wg.Done()
		for {
			val, ok := <-ch
			if !ok {
				break
			}

			fmt.Println("I am a go routine, and I got:", val.url)
		}
	}(results)

	wg.Wait()

	fmt.Println("Took: ", time.Since(start).Round(time.Millisecond))
}
