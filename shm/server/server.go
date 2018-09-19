/*

Modes:
 - HTTP: Send all data over HTTP POST.
 - SHM-HTTP: Send the request shared memory ID over HTTP, get back a response
     shared memory ID.
 - SHM: Send the request shared memory ID via sysv queue.
 - common-SHM: Server and client share the same space.
 */
package main

import (
	"github.com/gen2brain/shm"
	"fmt"
	"net/http"
	"log"
	"strconv"
	"io/ioutil"
	"github.com/siadat/ipc"
	"bytes"
	"os"
	"syscall"
	"strings"
	"net"
	"os/signal"
	"encoding/binary"
	"io"
	common "github.com/monktastic/go-learning/shm"
)



// Given a shm id, copies the memory into a new shm and returns the new id.
func respondToShm(reqId int) int {
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
	
	return respId
}

func httpShmHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitAfter(r.URL.Path, "/")
	reqId, _ := strconv.Atoi(parts[2])
	fmt.Printf("Reading from request id %d\n", reqId)
	
	respId := respondToShm(reqId)
	fmt.Printf("Writing to %d\n", respId)
	fmt.Fprintf(w, "%d", respId)
}


func httpHandler(w http.ResponseWriter, r *http.Request) {
	reqBytes, _ := ioutil.ReadAll(r.Body)
	fmt.Printf("Handling HTTP request of length %d\n", len(reqBytes))
	w.Write(reqBytes)
}

func main() {
	go queueShmAccept()
	go udsShmAccept()
	go udsAccept()
	go udsCommonShmAccept()
	http.HandleFunc("/shm/", httpShmHandler)
	http.HandleFunc("/http/", httpHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}


// Handles data over UDS
func udsAccept() {
	ln, err := net.Listen("unix", "/tmp/uds.sock")
	if err != nil {
		log.Fatal("Listen error: ", err)
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func(ln net.Listener, c chan os.Signal) {
		sig := <-c
		log.Printf("Caught signal %s: shutting down uds.", sig)
		ln.Close()
	}(ln, sigc)

	for {
		fd, err := ln.Accept()
		if err != nil {
			log.Fatal("Accept error: ", err)
		}
		go udsHandler(&fd)
	}
}

func udsHandler(fd *net.Conn) {
	sizeBytes := make([]byte, 4)
	(*fd).Read(sizeBytes)
	size := binary.LittleEndian.Uint32(sizeBytes)
	fmt.Printf("Receiving %d bytes\n", size)

	reqBytes := make([]byte, size)
	io.ReadFull(*fd, reqBytes)
	(*fd).Write(reqBytes)
	(*fd).Close()
}

// Uses UDS just to indicate that a request has come in (and to signal a
// response). Keeps the socket open.
func udsCommonShmAccept() {
	ln, err := net.Listen("unix", common.COMMON_SOCK)
	if err != nil {
		log.Fatal("Listen error: ", err)
	}

	// Close the socket on signals.
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func(ln net.Listener, c chan os.Signal) {
		sig := <-c
		log.Printf("Caught signal %s: shutting down %s", sig, common.COMMON_SOCK)
		ln.Close()
	}(ln, sigc)

	for i := 0; ; i++ {
		fd, err := ln.Accept()
		if err != nil {
			log.Fatal("Accept error: ", err)
			continue
		} else {
			fmt.Printf(
				"Accepted connection #%d on 'common' socket\n", i)
		}

		go udsCommonShmHandler(i, &fd)
	}
}

func udsCommonShmHandler(i int, fd *net.Conn) {
	// Create shm.
	socketName := fmt.Sprintf("%s-%d", common.COMMON_SOCK, i)
	os.Create(socketName)
	key, err := ipc.Ftok(socketName, 0)
	if err != nil {
		panic(err)
	}
	id, err := shm.Get(int(key), common.COMMON_SHM_SIZE, shm.IPC_CREAT|0777)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created common shm at key %d, id %d\n", key, id)
	shmData, err := shm.At(id, 0, 0)

	// Send the shm id.
	common.WriteInt(fd, uint32(id))
	
	for {
		reqSize, err := common.ReadInt(fd)
		if err != nil {
			log.Printf("Read failed: %s\n", err)
			break
		}
		fmt.Printf("Request size %d\n", reqSize)
		
		//req := make([]byte, reqSize)
		//copy(shmData[:reqSize], req)

		// Write the response size
		err = common.WriteInt(fd, reqSize)
		if err != nil {
			log.Fatal("Writing client error: ", err)
			break
		} else {
			fmt.Println("Wrote response size")
		}
	}

	(*fd).Close()
	os.Remove(socketName)
	shm.Dt(shmData)
	shm.Rm(id)
}


// Gets requests over UDS containing (new) shm id.
func udsShmAccept() {
	ln, err := net.Listen("unix", "/tmp/uds-shm.sock")
	if err != nil {
		log.Fatal("Listen error: ", err)
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func(ln net.Listener, c chan os.Signal) {
		sig := <-c
		log.Printf("Caught signal %s: shutting down uds/shm.", sig)
		ln.Close()
	}(ln, sigc)


	for {
		fd, err := ln.Accept()
		if err != nil {
			log.Fatal("Accept error: ", err)
		}
		go udsShmHandler(&fd)
	}
}

func udsShmHandler(fd *net.Conn) {
	buf := make([]byte, 4)
	_, err := (*fd).Read(buf)
	if err != nil {
		return
	}

	// Retrieve the shm id.
	reqShmId := binary.LittleEndian.Uint32(buf)
	fmt.Printf("Server got %d\n", reqShmId)

	respId := respondToShm(int(reqShmId))

	respBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(respBytes, uint32(respId))
	_, err = (*fd).Write(respBytes)
	if err != nil {
		log.Fatal("Writing client error: ", err)
	}

	(*fd).Close()
}

// Uses sysv queue to receive shm id.
func queueShmAccept() {
	fmt.Println("Started shm handler")
	_, err := os.Create("/tmp/server.8080")
	if err != nil {
		panic(err)
	}
	
	// Get the server key and queue id.
	key, err := ipc.Ftok("/tmp/server.8080", 0)
	if err != nil {
		panic(err)
	}

	var qid uint64
	for ;; {
		qid, err = ipc.Msgget(key, ipc.IPC_CREAT | ipc.IPC_EXCL | 0777)
		if err == nil {
			break
		} else if err == syscall.EEXIST {
			qid, err = ipc.Msgget(key, 0777)
			fmt.Printf("Deleting queue %d and re-creating\n", qid)
			err = ipc.Msgctl(qid, ipc.IPC_RMID)
			if err != nil {
				panic(fmt.Errorf("Could not delete queue %d", qid))
			}
		} else {
			panic(err)
		}
	}
	
	fmt.Printf("Opened queue at key %d, id %d\n", key, qid)

	for {
		rcvBuf := &ipc.Msgbuf{Mtype: 1}
		ipc.Msgrcv(qid, rcvBuf, 0)
		respBytes := rcvBuf.Mtext
		
		var reqMtype uint64
		var reqShmId int
		fmt.Fscanf(bytes.NewReader(respBytes), "%d %d", &reqMtype, &reqShmId)
		fmt.Printf("Read message: mtype %d shmid %d\n", reqMtype, reqShmId)

		go queueShmHandler(reqShmId, reqMtype, qid)
	}
}


func queueShmHandler(reqShmId int, reqMtype uint64, qid uint64) {
	respId := respondToShm(reqShmId)

	// Send null-terminated string.
	message := []byte(fmt.Sprintf("%d%c", respId, 0))
	msg := &ipc.Msgbuf{Mtype: reqMtype, Mtext: message}
	err := ipc.Msgsnd(qid, msg, 0)
	fmt.Printf("Sent response [%s]\n", message)
	if err != nil {
		panic(err)
	}
}