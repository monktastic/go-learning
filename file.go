package main

//import "fmt"
import "os"

func main() {
	file, _ := os.Create("file.txt")
	println(file)
}

