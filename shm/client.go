
package main

import (
	"github.com/gen2brain/shm"
	"strconv"
	"net/http"
	"bytes"
	"fmt"
	"io/ioutil"
	"sync"
	"time"
	"flag"
)

type requester func()
var MEM_SIZE int

func main() {
	nr := flag.Int("nr", 1000, "number of requests")
	nt := flag.Int("nt", 15, "number of threads")
	ms := flag.Int("bytes", 1<<25, "number of bytes per request")
	flag.Parse()
	MEM_SIZE = *ms
	
	fmt.Printf("%d %d %d\n", *nr, *nt, MEM_SIZE)

	fmt.Println("HTTP:")
	timeRequests(*nr, *nt, httpRequest)
	fmt.Println("SHM:")
	timeRequests(*nr, *nt, shmRequest)
}
	
func timeRequests(numReqs int, numWorkers int, fn func()) {
	var mu sync.Mutex // Protects the values.

	remaining := numReqs
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	start := time.Now()
	for i := 0; i < numWorkers; i++ {
	go worker(&mu, &wg, &remaining, shmRequest)
	}
	wg.Wait()
	fmt.Println()
	end := time.Now()
	elapsed := end.Sub(start)

	qps := 1.0E9 * float64(numReqs) / float64(elapsed)
	fmt.Printf("Time: %d ns; qps: %f\n", elapsed, qps)
}

func worker(mu *sync.Mutex, wg *sync.WaitGroup, remaining *int, fn func()) {
	defer wg.Done()
	for {
		mu.Lock()
		done := true 
		if (*remaining > 0) {
			done = false
			*remaining--
		}
		mu.Unlock()

		if done {
			return
		} else {
			fn()
		}
	}
}

func shmRequest() {
	reqId, err := shm.Get(shm.IPC_PRIVATE, MEM_SIZE, shm.IPC_CREAT|0777)
	if err != nil || reqId < 0 {
		panic(fmt.Sprintf("Could not shmget %d bytes", MEM_SIZE))
	}
	req_bytes, err := shm.At(reqId, 0, 0)
	if err != nil || reqId < 0 {
		panic(fmt.Sprintf("Could not shmat %d bytes", MEM_SIZE))
	}
	
	url := fmt.Sprintf("http://localhost:8080/%d", reqId)	
	//fmt.Printf("Calling %s\n", url)
	fmt.Print(".")
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	respIdBytes, _ := ioutil.ReadAll(resp.Body)
	respId, _ := strconv.Atoi(string(respIdBytes))
	respBytes, _ := shm.At(respId, 0, 0)

	defer func() {
		// Detach and delete response shm.
		shm.Dt(respBytes)
		shm.Rm(respId)
		// Detach request shm.
		shm.Dt(req_bytes)
	}()

	if (bytes.Compare(req_bytes, respBytes) != 0) {
		panic("Request and response were different!")
	}
}

func httpRequest() {
	reqBytes := make([]byte, MEM_SIZE)
	
	resp, _ := http.Post("http://localhost:8080/http", 
		"application/octet-stream", bytes.NewBuffer(reqBytes))
	defer resp.Body.Close()
	respBytes, _ := ioutil.ReadAll(resp.Body)
	
	if (bytes.Compare(reqBytes, respBytes) != 0) {
		panic("Request and response were different!")
	}
}