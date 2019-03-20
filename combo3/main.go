package main

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/gordonklaus/portaudio"
	"github.com/tarm/serial"
)

const pcMicSampleRate = 44100
const phoneMicSampleRate = 8000
const seconds = 0.01
const myIP = "192.168.25.18"
const connport = 2000
const secondIP = "192.168.25.32" //  "192.168.88.253"
const tcpPort = 8081
const baudrate = 9600
const uartPort = "/dev/cu.wchusbserial1410"

var fetchTime = time.Now
var lastCommand = ""

type Data struct {
	Command string
	Phone   string
	Text    string
}

func main() {

	debug := color.New(color.FgRed).PrintfFunc()

	/** INIT */
	portaudio.Initialize()
	defer portaudio.Terminate()

	c := &serial.Config{Name: uartPort, Baud: baudrate}
	uart, err := serial.OpenPort(c)
	if err != nil {
		fmt.Print(err)
	}

	/** THE UPSTREAM */

	upconn, err := net.Dial("udp", fmt.Sprintf("%s:%d", secondIP, connport))

	upbuffer := make([]int16, pcMicSampleRate*seconds)
	upstream, err := portaudio.OpenDefaultStream(1, 0, pcMicSampleRate, len(upbuffer), func(in []int16) {
		for i := range upbuffer {
			upbuffer[i] = in[i] //fmt.Sprintf("%f", in[i])
		}

		upconn.Write(int16ArrToByteArr(upbuffer))

		if err != nil {
			fmt.Printf("Some error %v", err)
			return
		}
	})
	chk(err)

	/** THE DOWNSTREAM */

	downbuffer := make([]byte, phoneMicSampleRate*seconds)
	p := make([]byte, phoneMicSampleRate*seconds*2)
	addr := net.UDPAddr{
		Port: connport,
		IP:   net.ParseIP(myIP),
	}
	ser, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Printf("Some error %v\n", err)
		return
	}
	downstream, err := portaudio.OpenDefaultStream(0, 1, phoneMicSampleRate, len(downbuffer), func(out []int16) {

		_, _, err := ser.ReadFromUDP(p)
		chk(err)
		buffOut := bytesToInt16Arr(p)

		for i := range out {
			out[i] = buffOut[i]
		}
	})

	chk(err)
	chk(upstream.Start())
	chk(downstream.Start())

	//TCP
	ln, _ := net.Listen("tcp", fmt.Sprintf(":%d", tcpPort))
	debug("Waiting for TCP connection...")
	conn, _ := ln.Accept()

	go recieve(conn, uart)

	// fmt.Fprintf(conn, "The network is")

	color.Green("AT")
	time.Sleep(time.Second)
	sendAT(uart, "AT")

	time.Sleep(time.Second)
	sendAT(uart, "AT+CREG?")

	for {
		time.Sleep(10 * time.Millisecond)
		// wg.Add(1)
		go checkTCP(conn, uart)
	}

	//DESTRUCTOR

	time.Sleep(time.Second * 60 * 30)

	color.Red("EXIT, CODE: 0.")
	defer uart.Close()
	defer upstream.Close()
	defer upconn.Close()
	chk(downstream.Stop())
	defer downstream.Close()

}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

func bytesToInt16Arr(in []byte) []int16 {
	var out [phoneMicSampleRate * seconds]int16
	for i := 0; i < len(in)/2; i++ {
		out[i] = bytesToInt16([]byte{in[i*2], in[i*2+1]})
	}
	return out[:]
}

func bytesToInt16(in []byte) int16 {
	// numBytes = []byte{0xf8, 0xe4}
	u := binary.BigEndian.Uint16(in)
	// fmt.Printf("%#X %[1]v\n", u) // 0XFF10 65296
	return int16(u)
}

func int16ArrToByteArr(in []int16) []byte {
	var out [pcMicSampleRate * seconds * 2]byte
	for i := 0; i < len(in); i++ {
		var part = int16ToBytes(in[i])
		out[i*2] = part[0]
		out[i*2+1] = part[1]
	}
	return out[:]
}

func int16ToBytes(i int16) []byte {
	// var out [2]byte
	var h, l uint8 = uint8(i >> 8), uint8(i & 0xff)
	return ([]byte{h, l})
}

func checkTCP(conn io.ReadWriteCloser, port io.ReadWriteCloser) {

	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	red2 := color.New(color.FgRed).Add(color.Underline).SprintFunc()

	message, err := bufio.NewReader(conn).ReadString('\n')

	if err == nil {

		//Getting data from Phone
		fmt.Printf("Message Received: %s", yellow(string(message)))
		var data Data
		err := json.Unmarshal([]byte(message), &data)
		if err != nil {
			fmt.Printf("%s %s", red("Problem decoding JSON"), red2(err))
		}
		fmt.Println(data.Command)

		//Working with data
		if strings.Compare(data.Command, "call") == 0 {
			fmt.Fprintf(port, "ATD+%s;\r\n", data.Phone)
		} else if strings.Compare(data.Command, "sendUSSD") == 0 {
			fmt.Fprintf(port, "AT+CUSD=1,\"%s\"\r\n", data.Text)
		} else if strings.Compare(data.Command, "sendSMS") == 0 {
			sendAT(port, "AT+CMGF=1")
			sendAT(port, fmt.Sprintf("AT+CMGS=\"%s\"", data.Phone))
			time.Sleep(time.Second)
			fmt.Fprintf(port, data.Text)
			time.Sleep(time.Second)
			fmt.Fprintf(port, string([]byte{0x1A}))
			sendAT(port, "AT+CMGF=0")
		} else if strings.Compare(data.Command, "AT") == 0 {
			sendAT(port, data.Text)
		} else if strings.Compare(data.Command, "status") == 0 {
			sendAT(port, "AT")
			sendAT(port, "AT+CREG?")
			sendAT(port, "AT+CSQ")
		} else if strings.Compare(data.Command, "epta") == 0 {
			fmt.Fprintf(port, string([]byte{0x1A}))
		} else if strings.Compare(data.Command, "readSMS") == 0 {
			if data.Text == "" {
				sendAT(port, "AT+CMGL=\"REC UNREAD\"")
			} else {
				fmt.Fprintf(port, "AT+CMGL=\"%s\"", data.Text)
			}
		}

	}

	// wg.Done()
}

func sendAT(s io.ReadWriteCloser, comm string) {
	color.Cyan(comm)
	_, err := s.Write([]byte(fmt.Sprintf("%s\r\n", comm)))
	if err != nil {
		fmt.Print(err)
	}
}

var voltageErrors = [...]string{"UNDER-VOLTAGE POWER DOWN", "UNDER-VOLTAGE WARNNING", "OVER-VOLTAGE POWER DOWN", "OVER-VOLTAGE WARNNING"}

func getResponse(conn net.Conn, in []byte) {
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	reO := regexp.MustCompile(`(.+)\n(.+)`)
	res0 := reO.FindAllString(string(in), -1)

	for _, item := range res0 {
		color.Cyan(hex.Dump([]byte(item)))
		// item = strings.TrimRight(item, "\n")
		color.Yellow("LAST: %s", item)

		for _, vE := range voltageErrors {
			if strings.Index(item, vE) > -1 {
				color.Red("Under-voltage")
				fmt.Fprintf(conn, red("{\"info\":\"Under-voltage\"}"))
			}
		}

		if strings.Index(item, "+CUSD:") > -1 {
			color.Yellow("The SMS was sent")
			fmt.Fprintf(conn, yellow("{\"SMS\":\"send\"}"))
		} else if strings.Index(item, "+CMTE") > -1 {
			color.Red("Device is running HOT")
			fmt.Fprintf(conn, red("{\"info\":\"Device is running HOT\"}"))
		} else if strings.Index(item, "\nNO CARRIER\r") > -1 {
			color.Red("Call ended")
			fmt.Fprintf(conn, green("{\"info\":\"call ended\"}"))
		} else if strings.Index(item, "+CUSD:") > -1 {
			re := regexp.MustCompile(`\+CUSD:\s\d+,\s\"(.+)\",\s\d+`)
			res := re.FindString(item)

			if res == "" {

			} else {
				color.Yellow("The USSD result is %s", res)
				fmt.Fprintf(conn, yellow("{\"USSD\":\"%s\"}"), res)
			}
		} else if strings.Index(item, "+CLIP:") > -1 {

			// color.Yellow("CLIP!!!!!!!!!!")

			re := regexp.MustCompile(`\+CLIP\:\s+\"(\+\d+)"`)
			res := re.FindStringSubmatch(item)

			if len(res) > 0 {
				if res[1] != "" {
					color.Yellow("Phone number %s", res[1])
					fmt.Fprintf(conn, yellow("{\"number\":\"%s\"}"), res[1])
				}
			}
		} else if strings.Index(item, "+CMTI:") > -1 {
			re := regexp.MustCompile(`\+CMTI:\s+\".+\",\d+`)
			res := re.FindString(item)

			if res == "" {

			} else {
				color.Yellow("New SMS %s", res)
				fmt.Fprintf(conn, yellow("{\"response\":\"new SMS\"}"), res)
			}
		} else if strings.Index(item, "+CSQ:") > -1 {
			re := regexp.MustCompile(`\+CSQ:\s+(\d+),`)
			rssi := re.FindStringSubmatch(item)

			if rssi[1] != "" {
				color.Yellow("RSSI %s", rssi[1])
				fmt.Fprintf(conn, yellow("{\"rssi\":%s}"), rssi[1])
			}

		} else if strings.Index(item, "+CREG:") > -1 {
			var (
				n    int
				stat int
			)
			_, err := fmt.Sscanf(item, "+CREG: %d,%d\r\n", &n, &stat)
			if err != nil {
				color.Red("Fscanf: %v\n", err)
			}
			color.Yellow("The network is %s", (map[bool]string{true: "connected", false: "searching"})[stat > 0])
			fmt.Fprintf(conn, yellow("{\"connected\":%s}"), (map[bool]string{true: "true", false: "false"})[stat > 0])
		} else if strings.Index(item, "ERROR") > -1 {
			color.Red("ERROR")
			fmt.Fprintf(conn, red("{\"response\":\"ERROR\"}"))
		} else {
			color.Red("LAST response is undefined")
			re := regexp.MustCompile(`\+CUSD:\s\d+,\s\"((?:.+|\n+|\r+)+)\",\s\d+`)
			res := re.FindStringSubmatch(lastCommand + item)
			if len(res) > 0 {
				if res[1] != "" {
					color.Yellow("The USSD result is %s", res[1])
					fmt.Fprintf(conn, yellow("{\"USSD\":%q}"), res[1])
				}
			}
		}
		lastCommand = item
	}

	if len(in) == 1 && in[0] == 0 {
		color.Red("Restart?!")
		fmt.Fprintf(conn, red("{\"response\":\"Restart?!\"}\n"))
	}
	// return out
}

func recieve(conn net.Conn, s io.ReadWriteCloser) {
	for {
		time.Sleep(time.Second)
		color.Cyan("READING...")
		buf := make([]byte, 256)
		n, err := s.Read(buf)
		if err != nil {
			fmt.Print(err)
		}
		getResponse(conn, buf[:n])
		color.Magenta(hex.Dump(buf[:n]))
	}
}
