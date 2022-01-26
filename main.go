package main

import (
	"runtime"
	"os"
        "os/exec"
	"fmt"
	"time"
	"log"
         "io"
         "strings"
         "bytes"
	"github.com/pin/tftp"
)


var Version string;
var Data string;

const (

cmdBOOT = "boot.bin"
cmdBrand = "brand.txt"
cmdFilenames = "filnames.txt"
cmdArtnetMode = "artnmod.txt"
cmdRGBMode = "rgbwmode.txt"
cmdCrop = "crop.txt"
cmdIP = "ip.txt"
cmdMAC = "mac.txt"
cmdStop = "stop.txt"
cmdReboot = "reboot.txt"
cmdTest = "dummy.txt"
)

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

       return Data+filename;  // example: /tmp  /some/file

    } else if filename[0] == '/' && Data[dataLen-1] == '/' {

       return Data+filename[1:]; // example: /tmp/  /some/file

    } else if filename[0] != '/' && Data[dataLen-1] == '/' {

       return Data+filename;  // example: /tmp/  some/file

    } else if filename[0] != '/' && Data[dataLen-1] != '/' {

       return Data+"/"+filename; // example: /tmp  some/file
    }

    return Data+filename;
}

// readHandler is called when client starts file download from server
func readHandler(filename string, rf io.ReaderFrom) error {

        fullfilename := createPath(filename)
	fmt.Printf("Query for %s\n", filename)
	fmt.Printf("Read from %s\n", fullfilename)

	file, err := os.Open(fullfilename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	        fmt.Printf("Open error %v\n", err)
		return err
	}
	n, err := rf.ReadFrom(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	        fmt.Printf("Read error %v\n", err)
		return err
	}
	fmt.Printf("%d bytes sent\n", n)
	return nil
}

func getIP(buf bytes.Buffer) (ip string, err error) {

    list := strings.Fields( buf.String() )
    if len(list) <= 0 {
       return "", fmt.Errorf("Incorrect format")
    }

    if list[0] != "IP:" {
       return "", fmt.Errorf("IP not found")
    }

    return list[1], nil
}

func changeIP(ip string) error {

	log.Printf("Change IP in progress, new ip: %s", ip)
	var buf bytes.Buffer
         //ifconfig eth0 1.2.3.4
	cmd := exec.Command("ifconfig", "eth0", ip)
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	if err != nil {
		return err
	}

	log.Printf("Successfully: %s", buf.String())
	return nil
}


func proceedCommand(cmd string, buf bytes.Buffer) error {

	fmt.Printf("Proceed command %s, with data: %v\n", cmd, buf)

	switch cmd {

	case cmdStop:
 	        fmt.Printf("Command 'STOP' found\n")
	case cmdIP:
 	        fmt.Printf("Command 'IP CHANGE' found\n")
                ip, err := getIP(buf)

                if err != nil {
                  return err
                }

                return changeIP(ip)

	case cmdBOOT:
 	        fmt.Printf("Command 'BOOT' found\n")
	case cmdBrand:
 	        fmt.Printf("Command 'BRAND' found\n")
	case cmdFilenames:
 	        fmt.Printf("Command 'FILE NAMES' found\n")
	case cmdArtnetMode:
 	        fmt.Printf("Command 'ARTNET MODE' found\n")
	case cmdRGBMode:
 	        fmt.Printf("Command 'RGB MODE' found\n")
	case cmdCrop:
 	        fmt.Printf("Command 'CROP' found\n")
	case cmdMAC:
 	        fmt.Printf("Command 'MAC' found\n")
	case cmdReboot:
 	        fmt.Printf("Command 'REBOOT' found\n")
	case cmdTest:
 	        fmt.Printf("Command 'TEST' found\n")
	default:
		return fmt.Errorf("Unregistered command '%s'",cmd)
	}


     return nil
}

// writeHandler is called when client starts file upload to server
func writeHandler(filename string, wt io.WriterTo) error {

        fullfilename := createPath(filename)
	fmt.Printf("\nQuery for %s\n", filename)
	fmt.Printf("Write to %s\n", fullfilename)

	err := os.MkdirAll(basePath(fullfilename), os.ModeDir|os.ModePerm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
                fmt.Printf("MkdirAll error %v\n", err)
		return err
	}

	file, err := os.Create(fullfilename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
                fmt.Printf("Open error %v\n", err)
		return err
	}

	var buf bytes.Buffer
	mw := io.MultiWriter(&buf, file)

	n, err := wt.WriteTo(mw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
                fmt.Printf("Write error %v\n", err)
		return err
	}
	fmt.Printf("%d bytes received\n", n)

	return proceedCommand(strings.ToLower(baseName(filename)), buf)
}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	fmt.Printf("Version is %s\n", Version)
	if len(os.Args) < 3 {
		log.Fatal("tftpx, incorrect number of arguments.\nUsage: " + os.Args[0] + " [ip]:port data_local_path")      //:69 
		log.Fatal("Example: %s: " + os.Args[0] + " :69 /tmp")      //:69 
	}

        fmt.Printf("Listening on %s\n", os.Args[1])
        fmt.Printf("Use local directory for data %s\n", os.Args[2])
        fmt.Printf("\n")

        Data = os.Args[2]

	// use nil in place of handler to disable read or write operations
	s := tftp.NewServer(readHandler, writeHandler)
	s.SetTimeout(5 * time.Second)            // optional
	err := s.ListenAndServe(os.Args[1]) // blocks until s.Shutdown() is called
	if err != nil {
		fmt.Fprintf(os.Stdout, "server: %v\n", err)
		os.Exit(1)
	}
}
