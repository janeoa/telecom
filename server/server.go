package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github.com/jacobsa/go-serial/serial"
)

type Data struct {
	Command string
	Phone   string
	Text    string
}

func main() {

	fmt.Println("connecting to UART...")
	options := serial.OpenOptions{
		// PortName:        "/dev/ttyS0",
		// PortName:        "/dev/tty.usbserial-1420",
		PortName:        "/dev/tty.wchusbserial1420",
		BaudRate:        115200,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 4,
	}
	port, err := serial.Open(options)
	if err != nil {
		log.Fatalf("serial.Open: %v", err)
	}

	sendUART(port, []byte{0x41, 0x54, 0x0D, 0x00})

	// port.Write([]byte{0x41, 0x54, 0x0D, 0x00})
	buf := make([]byte, 255)
	n, err := port.Read(buf)
	if err != nil {
		if err != io.EOF {
			fmt.Println("Error reading from serial port: ", err)
		}
	} else {
		buf = buf[:n]
		fmt.Println("Rx: ", decodeRX(hex.EncodeToString(buf)))
	}

	fmt.Println("Launching server...")

	// listen on all interfaces
	ln, _ := net.Listen("tcp", ":8081")
	fmt.Println("Waiting for TCP connection...")
	// accept connection on port
	conn, _ := ln.Accept()
	fmt.Println("Working...")

	defer port.Close()
	//
	// var wg sync.WaitGroup

	for {
		time.Sleep(100 * time.Milisecond())
		// wg.Add(1)
		go checkTCP(conn, port)
		// wg.Add(1)
		go readUARTtoBuff(port, buf)
		// wg.Wait()
		// color.Blue("Checking TCP")
	}

	//
}

func checkTCP(conn io.ReadWriteCloser, port io.ReadWriteCloser) {

	// color.Blue("Checking TCP")

	message, err := bufio.NewReader(conn).ReadString('\n')

	if err == nil {

		//Getting data from Phone
		fmt.Print("Message Received:", string(message))
		var data Data
		err := json.Unmarshal([]byte(message), &data)
		if err != nil {
			fmt.Println("Problem decoding JSON ", err)
		}
		fmt.Println(data.Command)

		//Working with data
		if strings.Compare(data.Command, "call") == 0 {
			call(port, data.Phone)
		} else if strings.Compare(data.Command, "sendUSSD") == 0 {
			sendUSSD(port, data.Text)
		} else if strings.Compare(data.Command, "sendSMS") == 0 {

		}

	}

	// wg.Done()
}

func sendUART(port io.ReadWriteCloser, comm []byte) {
	// b := []byte{0x43, 0x75, 0x6E, 0x74}
	_, err := port.Write(comm)
	// n, err := port.Write(b)
	if err != nil {
		log.Fatalf("port.Write: %v", err)
	}
	// fmt.Println("Wrote", n, "bytes.")
}

func call(port io.ReadWriteCloser, phone string) {
	fmt.Println("calling... ", phone)
	// fmt.Println(fmt.Sprintf("ATD%s;", phone))
	fmt.Fprintf(port, "ATD+77058400077;")
	// n, err := port.Write(b)
	// if err != nil {
	// 	log.Fatalf("port.Write: %v", err)
	// }
	// fmt.Println("Wrote", n, "bytes.")
}

func sendUSSD(port io.ReadWriteCloser, command string) {
	fmt.Println("Sending USSD... ")
	// b := []byte{0x41, 0x54, 0x0D, 0x00} //
	// n, err := port.Write(b)
	// fmt.Fprintf(port, "AT+CUSD=1\n") //AT+CUSD=1,\"*111#\"\n
	// time.Sleep(1000 * time.Millisecond)
	fmt.Fprintf(port, "AT+CUSD=1,\"*111#\"\n")

	fmt.Println("Command send")
}

func readUARTtoBuff(port io.ReadWriteCloser, buf []byte) {

	// color.Magenta("READING UART")

	n, err := port.Read(buf)
	// fmt.Println("\t\t#####\t the lenght is ", n, "\t#####\t\t")
	if err != nil {
		if err != io.EOF {
			fmt.Println("Error reading from serial port: ", err)
		}
	} else {
		buf = buf[:n]
		// fmt.Println("Rx: ", decodeRX(hex.EncodeToString(buf)))
		fmt.Print(decodeRX(hex.EncodeToString(buf)))

	}

	// wg.Done()

	// return n
}

func decodeRX(raw string) string {
	out, err := hex.DecodeString(raw)
	if err != nil {
		fmt.Println(err)
	}
	return string(out)
}
