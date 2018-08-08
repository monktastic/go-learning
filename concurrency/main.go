package main

import (
	"fmt"
	"net/http"
	"sync"
	"log"
	"io/ioutil"
	"strconv"
)

/*
  First run add_one_server.go, and then run this.
 */
func main() {
	NUM_TASKS := 256
	NUM_STATES := 1000

	states := make([]int, NUM_STATES)
	for i := 0; i < NUM_STATES; i++ {
		states[i] = i;
	}

	values := make([]int, len(states))

	var mu sync.Mutex // Protects the values.
	ch := make(chan int, NUM_TASKS)

	var wg sync.WaitGroup
	wg.Add(NUM_TASKS)
	for i := 0; i < NUM_TASKS; i++ {
		go runMCTSWorker(values, ch, &mu, &wg)
	}

	for i := range states {
		ch <- i
	}
	close(ch)
	wg.Wait()

	// Print all values
	for i := 0; i < NUM_STATES; i++ {
		fmt.Printf("%d ", values[i])
	}
	
	fmt.Println("Done!")
}

func runMCTSWorker(
	values []int,
	ch <- chan int, mu *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		i, open := <- ch
		if !open {
			break
		}
		i_str := strconv.Itoa(i)
		v := remoteCall("http://localhost:9000/" + i_str)

		mu.Lock()
		values[i] = v
		mu.Unlock()
	}
}

func remoteCall(url string) (int) {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
		return -1;
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
		return -1;
	}
	val, _ := strconv.ParseInt(string(body), 10, 32)
	return int(val)
}
