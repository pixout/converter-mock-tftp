package main

import (
	"bytes"
	"fmt"
	"github.com/pin/tftp"
	"io"
	"log"
	"log/syslog"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type messageQueue chan func()

var Version string
var Data string
var mq messageQueue

const (
	cmdBOOT       = "boot.bin"
	cmdBrand      = "brand.txt"
	cmdFilenames  = "filnames.txt"
	cmdArtnetMode = "artnmod.txt"
	cmdRGBMode    = "rgbwmode.txt"
	cmdCrop       = "crop.txt"
	cmdIP         = "ip.txt"
	cmdMAC        = "mac.txt"
	cmdStop       = "stop.txt"
	cmdReboot     = "reboot.txt"
	cmdTest       = "dummy.txt"
)

func (mq messageQueue) enqueue(f func()) {
	if len(mq) >= 9 {
		log.Printf("job discarded, buffer full: %d\n", len(mq))
		return
	}

	mq <- f
}

func baseName(s string) string {
	n := strings.LastIndexByte(s, '/')
	if n == -1 {
		return s
	}
	return s[n+1:]
}

func basePath(s string) string {
	n := strings.LastIndexByte(s, '/')
	if n == -1 {
		return s
	}
	return s[:n]
}

func createPath(filename string) string {

	dataLen := len(Data)

	if filename[0] == '/' && Data[dataLen-1] != '/' {

		return Data + filename // example: /tmp  /some/file

	} else if filename[0] == '/' && Data[dataLen-1] == '/' {

		return Data + filename[1:] // example: /tmp/  /some/file

	} else if filename[0] != '/' && Data[dataLen-1] == '/' {

		return Data + filename // example: /tmp/  some/file

	} else if filename[0] != '/' && Data[dataLen-1] != '/' {

		return Data + "/" + filename // example: /tmp  some/file
	}

	return Data + filename
}

// readHandler is called when client starts file download from server
func readHandler(filename string, rf io.ReaderFrom) error {

	fullfilename := createPath(filename)
	log.Printf("Query for %s\n", filename)
	log.Printf("Read from %s\n", fullfilename)

	file, err := os.Open(fullfilename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		log.Printf("Open error %v\n", err)
		return err
	}
	n, err := rf.ReadFrom(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		log.Printf("Read error %v\n", err)
		return err
	}
	log.Printf("%d bytes sent\n", n)
	return nil
}

func getIP(buf bytes.Buffer) (ip, mask string, err error) {

	list := strings.Fields(buf.String())
	if len(list) <= 0 {
		return "", "", fmt.Errorf("Incorrect format")
	}

	if list[0] != "IP:" {
		return "", "", fmt.Errorf("IP not found")
	}

	if list[2] != "NETMASK:" {
		return "", "", fmt.Errorf("MASK not found")
	}

	return list[1], list[3], nil
}

func changeIP(ip, mask string) error {

	log.Printf("Trying to change IP/MASK to %s/%s", ip, mask)

	_, err := exec.LookPath("ifconfig")
	if err != nil {
		log.Printf("%v", err)
		return err
	}

	log.Printf("ifconfig - found, scheduling job till 'REBOOT' command")

	//todo: store go routines inside buffer and execue by reset, otherwise it will not work
	mq.enqueue(func() {

		var buf bytes.Buffer
		//ifconfig eth0 1.2.3.4
		cmd := exec.Command("ifconfig", "eth0", ip, "netmask", mask)
		cmd.Stdout = &buf
		cmd.Stderr = &buf

		err := cmd.Run()
		if err != nil {
			log.Printf("Unsuccessfully! IP not changed")
			return
		}

		log.Printf("Successfully: %s", buf.String())
	})

	return nil
}

func proceedCommand(cmd string, buf bytes.Buffer) error {

	log.Printf("Proceed command %s, with data: %v\n", cmd, buf)

	switch cmd {

	case cmdStop:
		log.Printf("Command 'STOP' found\n")
	case cmdIP:
		log.Printf("Command 'IP CHANGE' found\n")
		ip, mask, err := getIP(buf)

		if err != nil {
			return err
		}

		return changeIP(ip, mask)

	case cmdBOOT:
		log.Printf("Command 'BOOT' found\n")
	case cmdBrand:
		log.Printf("Command 'BRAND' found\n")
	case cmdFilenames:
		log.Printf("Command 'FILE NAMES' found\n")
	case cmdArtnetMode:
		log.Printf("Command 'ARTNET MODE' found\n")
	case cmdRGBMode:
		log.Printf("Command 'RGB MODE' found\n")
	case cmdCrop:
		log.Printf("Command 'CROP' found\n")
	case cmdMAC:
		log.Printf("Command 'MAC' found\n")
	case cmdReboot:
		log.Printf("Command 'REBOOT' found\n")

		delay, _ := time.ParseDuration("10ms")

		log.Printf("Scheduled job queue len: %d\n", len(mq))

		time.AfterFunc(delay, func() {
			for f := range mq {
				f()
			}
		})

	case cmdTest:
		log.Printf("Command 'TEST' found\n")
	default:
		return fmt.Errorf("Unregistered command '%s'", cmd)
	}

	return nil
}

// writeHandler is called when client starts file upload to server
func writeHandler(filename string, wt io.WriterTo) error {

	fullfilename := createPath(filename)
	log.Printf("Query for %s\n", filename)
	log.Printf("Write to %s\n", fullfilename)

	err := os.MkdirAll(basePath(fullfilename), os.ModeDir|os.ModePerm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		log.Printf("MkdirAll error %v\n", err)
		return err
	}

	file, err := os.Create(fullfilename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		log.Printf("Open error %v\n", err)
		return err
	}

	var buf bytes.Buffer
	mw := io.MultiWriter(&buf, file)

	n, err := wt.WriteTo(mw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		log.Printf("Write error %v\n", err)
		return err
	}
	log.Printf("%d bytes received\n", n)

	return proceedCommand(strings.ToLower(baseName(filename)), buf)
}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	mq = make(messageQueue, 10)

	fmt.Printf("Version is %s\n", Version)
	if len(os.Args) < 3 {
		fmt.Printf("tftpx, incorrect number of arguments.\nUsage: " + os.Args[0] + " [ip]:port data_local_path") //:69
		fmt.Printf("Example: %s: " + os.Args[0] + " :69 /tmp")                                                   //:69
		os.Exit(1)
	}

	fmt.Printf("Listening on %s\n", os.Args[1])
	fmt.Printf("Use local directory for data %s\n", os.Args[2])
	fmt.Printf("\n")

	Data = os.Args[2]

	if runtime.GOOS[:3] == "win" {
		fmt.Println("Windows OS detected")
	} else if runtime.GOOS[:3] == "lin" {

		// Configure logger to write to the syslog.
		logwriter, err := syslog.New(syslog.LOG_NOTICE, baseName(os.Args[0]))
		if err == nil {
			log.SetOutput(logwriter)
		}

		fmt.Println("Linux OS detected")

	}

	// use nil in place of handler to disable read or write operations
	s := tftp.NewServer(readHandler, writeHandler)
	s.SetTimeout(5 * time.Second)       // optional
	err := s.ListenAndServe(os.Args[1]) // blocks until s.Shutdown() is called
	if err != nil {
		fmt.Fprintf(os.Stdout, "server: %v\n", err)
		os.Exit(1)
	}
}
