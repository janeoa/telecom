package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	"github.com/jacobsa/go-serial/serial"
)

type Data struct {
	Command string
	Phone   string
	Text    string
}

func sendUART(port io.ReadWriteCloser, comm []byte) {
	b := []byte{0x43, 0x75, 0x6E, 0x74}
	n, err := port.Write(b)
	if err != nil {
		log.Fatalf("port.Write: %v", err)
	}

	fmt.Println("Wrote", n, "bytes.")
}

func main() {

	fmt.Println("connecting to UART...")
	options := serial.OpenOptions{
		// PortName:        "/dev/ttyS0",
		// PortName:        "/dev/tty.usbserial-1420",
		PortName:        "/dev/tty.wchusbserial1420",
		BaudRate:        9600,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 4,
	}
	port, err := serial.Open(options)
	if err != nil {
		log.Fatalf("serial.Open: %v", err)
	}

	sendUART(port, []byte{0x43, 0x75, 0x6E, 0x74})

	fmt.Println("Launching server...")

	// listen on all interfaces
	ln, _ := net.Listen("tcp", ":8081")
	fmt.Println("is it waiting? 1")
	// accept connection on port
	conn, _ := ln.Accept()
	fmt.Println("is it waiting? 2")

	for {
		message, err := bufio.NewReader(conn).ReadString('\n')

		if err == nil {
			fmt.Print("Message Received:", string(message))
			var data Data

			err := json.Unmarshal([]byte(message), &data)
			if err != nil {
				fmt.Println("Problem decoding JSON ", err)
			}
			fmt.Println(data.Command)
			if strings.Compare(data.Command, "call") == 0 {
				fmt.Println("calling... ", data.Phone)
				b := []byte{0x41, 0x54, 0x0D, 0x00} //
				n, err := port.Write(b)
				if err != nil {
					log.Fatalf("port.Write: %v", err)
				}

				fmt.Println("Wrote", n, "bytes.")
			}

		} else {
			fmt.Print("Connection closed\n")
			defer port.Close()
			os.Exit(0)

		}
	}
}
