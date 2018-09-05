
package main

import (
	"github.com/gen2brain/shm"
	"fmt"
	"net/http"
	"log"
	"strconv"
	"io/ioutil"
)

func shmHandler(w http.ResponseWriter, r *http.Request) {
	reqId, _ := strconv.Atoi(r.URL.Path[1:])
	fmt.Printf("Reading from request id %d\n", reqId)
	reqBytes, _ := shm.At(reqId, 0, 0)

	shmSize := len(reqBytes)
	//fmt.Printf("Num bytes is %d\n", shmSize)
	respId, err := shm.Get(shm.IPC_PRIVATE, shmSize, shm.IPC_CREAT|0777)
	if err != nil || respId < 0 {
		panic(fmt.Sprintf("Could not shmget %d bytes", shmSize))
	}
	respBytes, err := shm.At(respId, 0, 0)
	if err != nil || respId < 0 {
		panic(fmt.Sprintf("Could not shmat %d bytes", shmSize))
	}

	copy(respBytes[:], reqBytes)

	// Detach and delete request shm.
	shm.Dt(reqBytes)
	shm.Rm(reqId)
	// Detach response.
	shm.Dt(respBytes)
	
	fmt.Printf("Writing to %d\n", respId)
	fmt.Fprintf(w, "%d", respId)
}


func httpHandler(w http.ResponseWriter, r *http.Request) {
	reqBytes, _ := ioutil.ReadAll(r.Body)
	respBytes := make([]byte, len(reqBytes))

	copy(respBytes[:], reqBytes)
	fmt.Fprint(w, respBytes)
}

func main() {
	http.HandleFunc("/", shmHandler)
	http.HandleFunc("/http", httpHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

