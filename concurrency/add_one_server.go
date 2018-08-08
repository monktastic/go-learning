package main

import (
	"net/http"
	"strings"
	"strconv"
	"log"
)

func sayHello(w http.ResponseWriter, r *http.Request) {
	message := r.URL.Path
	log.Print(message)
	message = strings.TrimPrefix(message, "/")
	val, _ := strconv.ParseInt(message, 10, 32)
	val += 1
	message = strconv.FormatInt(val, 10)
	w.Write([]byte(message))
}

func main() {
	http.HandleFunc("/", sayHello)
	if err := http.ListenAndServe(":9000", nil); err != nil {
		panic(err)
	}
}