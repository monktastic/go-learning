
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
	"github.com/siadat/ipc"
	"os"
	"math/rand"
	"net"
	"log"
	"encoding/binary"
)

type requester func()
var MEM_SIZE int

var COMMON_SHM_SIZE = 1<<25
var COMMON_SOCK = "/tmp/uds-common-shm.sock"

var CLIENT_MEMORY []byte


func main() {
	nr := flag.Int("nr", 1000, "number of requests")
	nt := flag.Int("nt", 15, "number of threads")
	ms := flag.Int("bytes", 1<<25, "number of bytes per request")
	flag.Parse()
	MEM_SIZE = *ms
	
	fmt.Printf("%d %d %d\n", *nr, *nt, MEM_SIZE)

	/////////
	if (*nt == 1 && MEM_SIZE <= COMMON_SHM_SIZE) {
		fmt.Println("Common SHM over UDS:")
		conn, err := net.Dial("unix", COMMON_SOCK)
		if err != nil {
			log.Fatal("Dial error", err)
		}
		//defer conn.Close()

		key, err := ipc.Ftok(COMMON_SOCK, 0)
		if err != nil {
			panic(err)
		}
		if key <= 0 {
			panic(fmt.Sprintf("Bad key: %d\n", key))
		}
		id, err := shm.Get(int(key), COMMON_SHM_SIZE, 0777)
		shmBytes, err := shm.At(id, 0, 0)
		if err != nil {
			panic(err)
		}

		f := getUdsCommonShmRequest(conn, shmBytes)
		timeRequests(*nr, *nt, f)
	} else {
		fmt.Println("(Must have 1 thread and bytes < COMMON_SHM_SIZE " +
			"to run common SHM)")
	}


	CLIENT_MEMORY = make([]byte, MEM_SIZE)

	/////////
	fmt.Println("UDS:")
	timeRequests(*nr, *nt, udsRequest)

	/////////
	fmt.Println("HTTP:")
	timeRequests(*nr, *nt, httpRequest)

	
	/////////
	fmt.Println("SHM over Q:")
	timeRequests(*nr, *nt, queueShmRequest)

	/////////
	fmt.Println("SHM over UDS:")
	timeRequests(*nr, *nt, udsShmRequest)

	/////////
	fmt.Println("SHM over HTTP:")
	timeRequests(*nr, *nt, httpShmRequest)

	/////////////////////

	
	fmt.Println("Done")
}


func getUdsCommonShmRequest(conn net.Conn, shmBytes []byte) func() {
	return func() {
		//reqBytes := make([]byte, MEM_SIZE)
		//copy(shmBytes, reqBytes)
	
		// Tell server the request length.
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(MEM_SIZE))
		_, err := conn.Write(buf)
		if err != nil {
			panic(err)
		}
		
		_, err = conn.Read(buf)
		respByteSize := binary.LittleEndian.Uint32(buf)
		if err != nil {
			panic(err)
		}
		if (int(respByteSize) != MEM_SIZE) {
			panic(fmt.Sprintf("Response size is %d, not %d", 
				respByteSize, MEM_SIZE))
		}
		//if (bytes.Compare(reqBytes, shmBytes[:MEM_SIZE]) != 0) {
		//	panic("Request and response were different!")
		//}
	}
}

func timeRequests(numReqs int, numWorkers int, fn func()) {
	var mu sync.Mutex // Protects the values.

	remaining := numReqs
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	start := time.Now()
	for i := 0; i < numWorkers; i++ {
		go worker(&mu, &wg, &remaining, fn)
	}
	wg.Wait()
	fmt.Println()
	end := time.Now()
	elapsed := end.Sub(start)

	qps := 1.0E9 * float64(numReqs) / float64(elapsed)
	fmt.Printf("Time: %d ns; qps: %f\n", elapsed, qps)
	fmt.Printf("mspq (ms per query): %f ms\n", 1000.0/qps)
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
			fmt.Print(".")
		}
	}
}

func getShm() (int, []byte) {
	// Share the memory.
	reqId, err := shm.Get(shm.IPC_PRIVATE, MEM_SIZE, shm.IPC_CREAT|0777)
	if err != nil || reqId < 0 {
		panic(fmt.Sprintf("Could not shmget %d bytes", MEM_SIZE))
	}
	reqBytes, err := shm.At(reqId, 0, 0)
	if err != nil || reqId < 0 {
		panic(fmt.Sprintf("Could not shmat %d bytes", MEM_SIZE))
	}
	return reqId, reqBytes
}

func queueShmRequest() {
	reqId, reqBytes := getShm()

	// Get the server key and queue id.
	key, err := ipc.Ftok("/tmp/server.8080", 0)
	if err != nil {
		panic(err)
	}
	qid, err := ipc.Msgget(key, 0777)

	reqMtype := uint64(os.Getpid() << 32) + uint64(rand.Int31())
	message := []byte(fmt.Sprintf("%d %d%c", reqMtype, reqId, 0))
	msg := &ipc.Msgbuf{Mtype: 1, Mtext: message}
	//fmt.Printf("Opening queue at key %d, id %d\n", key, qid)
	//fmt.Printf("Sending message [%s]\n", msg.Mtext)
	err = ipc.Msgsnd(qid, msg, 0)
	if err != nil {
		panic(err)
	}
	
	rcvBuf:= &ipc.Msgbuf{Mtype: reqMtype, Mtext: []byte("")}
	ipc.Msgrcv(qid, rcvBuf, 0)
	respIdBytes := rcvBuf.Mtext
	//fmt.Printf("Response id bytes [%s]\n", respIdBytes)
	var respId int
	fmt.Fscanf(bytes.NewReader(respIdBytes), "%d", &respId)
	//fmt.Printf("Response id %d\n", respId)
	respBytes, _ := shm.At(respId, 0, 0)
	
	defer func() {
		// Detach and delete response shm.
		shm.Dt(respBytes)
		shm.Rm(respId)
		// Detach request shm.
		shm.Dt(reqBytes)
	}()

	//if (bytes.Compare(reqBytes, respBytes) != 0) {
	//	panic("Request and response were different!")
	//}
}

func httpShmRequest() {
	reqId, reqBytes := getShm()

	url := fmt.Sprintf("http://localhost:8080/shm/%d", reqId)	
	resp, _ := http.Get(url)
	if resp.StatusCode != 200 {
		panic(fmt.Sprintf("Bad response: %s", resp.Status))
	}
	defer resp.Body.Close()
	respIdBytes, _ := ioutil.ReadAll(resp.Body)
	respId, _ := strconv.Atoi(string(respIdBytes))
	respBytes, _ := shm.At(respId, 0, 0)

	defer func() {
		// Detach and delete response shm.
		shm.Dt(respBytes)
		shm.Rm(respId)
		// Detach request shm.
		shm.Dt(reqBytes)
	}()

	//if (bytes.Compare(reqBytes, respBytes) != 0) {
	//	panic("Request and response were different!")
	//}
}

func httpRequest() {
	//reqBytes := make([]byte, MEM_SIZE)

	resp, _ := http.Post("http://localhost:8080/http/", "", 
		bytes.NewBuffer(CLIENT_MEMORY))
	defer resp.Body.Close()
	ioutil.ReadAll(resp.Body)
	//if (bytes.Compare(reqBytes, respBytes) != 0) {
	//	panic("Request and response were different!")
	//}
}


func udsShmRequest() {
	reqId, reqBytes := getShm()
	
	conn, err := net.Dial("unix", "/tmp/uds-shm.sock")
	if err != nil {
		log.Fatal("Dial error", err)
	}
	defer conn.Close()
	
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(reqId))
	conn.Write(buf)
	if err != nil {
		log.Fatal("Write error:", err)
		return
	}
	_, err = conn.Read(buf)
	if err != nil {
		panic(err)
	}
	respId := binary.LittleEndian.Uint32(buf)
	respBytes, _ := shm.At(int(respId), 0, 0)

	defer func() {
		// Detach and delete response shm.
		shm.Dt(respBytes)
		shm.Rm(int(respId))
		// Detach request shm.
		shm.Dt(reqBytes)
	}()

	//if (bytes.Compare(reqBytes, respBytes) != 0) {
	//	panic("Request and response were different!")
	//}
}

func udsRequest() {
	conn, err := net.Dial("unix", "/tmp/uds.sock")
	if err != nil {
		log.Fatal("Dial error", err)
	}
	defer conn.Close()
	
	bs := make([]byte, 4)
	binary.LittleEndian.PutUint32(bs, uint32(MEM_SIZE))
	conn.Write(bs)
	
	_, err = conn.Write(CLIENT_MEMORY)
	if err != nil {
		panic(err)
	}

	ioutil.ReadAll(conn)
	//if (bytes.Compare(reqBytes, respBytes) != 0) {
	//	panic("Request and response were different!")
	//}
}