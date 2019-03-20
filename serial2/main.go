package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/fatih/color"
	"github.com/tarm/serial"
)

func main() {
	c := &serial.Config{Name: "/dev/cu.wchusbserial1420", Baud: 9600}
	s, err := serial.OpenPort(c)
	if err != nil {
		fmt.Print(err)
	}

	go recieve(s)

	time.Sleep(time.Second)
	sendAT(s, "AT")

	time.Sleep(time.Second)
	sendAT(s, "AT+CREG?")

	time.Sleep(time.Second * 10)
	color.Red("EXIT, CODE: 0.")
}

func sendAT(s io.ReadWriteCloser, comm string) {
	color.Cyan(comm)
	_, err := s.Write([]byte(fmt.Sprintf("%s\r\n", comm)))
	if err != nil {
		fmt.Print(err)
	}
}

func recieve(s io.ReadWriteCloser) {
	for {
		time.Sleep(time.Second)
		color.Cyan("READING...")
		buf := make([]byte, 100)
		n, err := s.Read(buf)
		if err != nil {
			fmt.Print(err)
		}
		fmt.Printf("%s", hex.Dump(buf[:n]))
	}
}
