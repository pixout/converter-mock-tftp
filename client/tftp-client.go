package main

import (
	"fmt"
	"github.com/pin/tftp"
	"log"
	"os"
	"runtime"
	"time"
)

var Version string

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	fmt.Printf("Version %s\n", Version)
	if len(os.Args) < 3 {
		log.Fatal("tftpx, incorrect number of arguments.\nUsage: " + os.Args[0] + " [ip]:port file_local_path file_remote_path")
	}

	path := os.Args[2]
	c, err := tftp.NewClient(os.Args[1])

	if err != nil {
		log.Fatal("Connection error: ", err)
	}

	file, err := os.Open(path)

	if err != nil {
		log.Fatal("Open error: ", err)
	}

	c.SetBlockSize(512)
	c.SetTimeout(5 * time.Second) // optional
	c.SetRetries(3)
	rf, err := c.Send(os.Args[3], "octet")

	if err != nil {
		log.Fatal("Send error: ", err)
	}

	n, err := rf.ReadFrom(file)

	if err != nil {
		log.Fatal("Read error: ", err)
	}

	fmt.Printf("Success, %d bytes sent\n", n)
	os.Exit(0)
}
