package main

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/gordonklaus/portaudio"
	"github.com/tarm/serial"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

const pcMicSampleRate = 8000     //44100
const phoneMicSampleRate = 44100 //8000
const seconds = 0.01

// const myIP = "192.168.0.100"
var myIP = "192.168.25.17" //default (changes)
var conn net.Conn
var fetchTime = time.Now
var cusdFlag = false
var cusd string

const connport = 2000
const secondIP = "192.168.25.45" //  "192.168.88.253"
const tcpPort = 8081
const baudrate = 9600
const uartPort = "/dev/cu.usbmodem14201"

// var multiline = false

// var ln net.Listener
// var conn net.Conn

// Data is for JSON
type Data struct {
	Command string
	Phone   string
	Text    string
}

// UCS2 is to convert
type UCS2 []byte

func main() {

	ip, err := externalIP()
	if err != nil {
		color.Red("%s", err)
		color.Red("Because of error, default ip is used")
	} else {
		if myIP != ip {
			color.Red("The ip is changed!\n")
		}
		myIP = ip
	}
	color.Green("local    ip is %s\n", myIP)
	color.Green("External ip is %s\n", secondIP)

	// debug := color.New(color.FgRed).PrintfFunc()

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

	go tcpLoop(&conn, uart)

	go recieve(&conn, uart)

	// fmt.Fprintf(conn, "The network is")

	color.Green("AT")
	time.Sleep(time.Second)
	sendAT(uart, "AT")

	time.Sleep(time.Second)
	sendAT(uart, "AT+CSQ")

	time.Sleep(time.Second)
	sendAT(uart, "AT+CREG?")

	time.Sleep(time.Second)
	sendAT(uart, "AT+CLIP=1")

	time.Sleep(time.Second)
	sendAT(uart, "AT+CVHU=0")

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

func checkTCP(conn *net.Conn, port io.ReadWriteCloser) {

	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	red2 := color.New(color.FgRed).Add(color.Underline).SprintFunc()

	for {
		message, err := bufio.NewReader(*conn).ReadString('\n')

		if err == nil {

			color.Green(message)
			//Getting data from Phone
			fmt.Printf("Message Received: %s", yellow(string(message)))
			var data Data
			err := json.Unmarshal([]byte(message), &data)
			if err != nil {
				fmt.Printf("%s %s", red("Problem decoding JSON"), red2(err))
			}
			fmt.Println(data.Command)

			if len(data.Phone) > 1 {
				data.Phone = formatPhone(data.Phone)
			}
			color.Red(data.Phone)
			//Working with data
			if strings.Compare(data.Command, "call") == 0 {
				fmt.Fprintf(port, "ATD%s;\r\n", data.Phone)
			} else if strings.Compare(data.Command, "sendUSSD") == 0 {
				fmt.Fprintf(port, "AT+CUSD=1,\"%s\", 15\r\n", data.Text)
			} else if strings.Compare(data.Command, "sendSMS") == 0 {
				var res []rune
				out := "0001000B91"
				phone := encodePhone(data.Phone)
				res = append(res, []rune(out + phone + "00" + "08")[:]...)
				// res := out + phone + "00" + "08" + "AA"
				text := hex.EncodeToString(UCS2.Encode([]byte(data.Text)))
				hexL := strconv.FormatInt(int64(len(text)/2), 16)
				if len(hexL) < 2 {
					hexL = "0" + hexL
				}

				res = append(res, []rune(hexL)[:]...)
				res = append(res, []rune(text)[:]...)

				lengz := len(res)/2 - 1

				color.Yellow("Lengz: %d", lengz)

				fmt.Fprintf(port, "AT+CMGS=%d\r", lengz)
				time.Sleep(time.Second)
				fmt.Fprintf(port, "%s\r", string(res))
				time.Sleep(time.Second)
				fmt.Fprintf(port, string([]byte{0x1A}))
				// sendAT(port, "AT+CMGF=1")
				// sendAT(port, fmt.Sprintf("AT+CMGS=\"%s\"", data.Phone))
				// time.Sleep(time.Second)
				// fmt.Fprintf(port, data.Text)
				// time.Sleep(time.Second)
				// fmt.Fprintf(port, string([]byte{0x1A}))
				// sendAT(port, "AT+CMGF=0")

			} else if strings.Compare(data.Command, "AT") == 0 {
				sendAT(port, data.Text)
			} else if strings.Compare(data.Command, "status") == 0 {
				sendAT(port, "AT+CREG?")
			} else if strings.Compare(data.Command, "rssi") == 0 {
				sendAT(port, "AT+CSQ")
			} else if strings.Compare(data.Command, "acceptCall") == 0 {
				sendAT(port, "ATA")
			} else if strings.Compare(data.Command, "callEnd") == 0 {
				sendAT(port, "ATH")
			} else if strings.Compare(data.Command, "readSMS") == 0 {
				if data.Text == "" {
					sendAT(port, "AT+CMGL=\"REC UNREAD\"")
				} else {
					fmt.Fprintf(port, "AT+CMGL=\"%s\"", data.Text)
				}
			}

		} else {
			fmt.Println(err)
			if err == io.EOF {
				// conn, _ = ln.Accept()
				fmt.Println("Error reading:", err.Error())
				yo := *conn
				yo.Close()
				break
			}
		}
	}
	// wg.Done()
}

func sendAT(s io.ReadWriteCloser, comm string) {
	color.Cyan("sendAT: %s", comm)
	_, err := s.Write([]byte(fmt.Sprintf("%s\r\n", comm)))
	if err != nil {
		fmt.Print(err)
	}
}

var voltageErrors = [...]string{"UNDER-VOLTAGE POWER DOWN", "UNDER-VOLTAGE WARNNING", "OVER-VOLTAGE POWER DOWN", "OVER-VOLTAGE WARNNING"}
var lastCommand []byte

func getResponse(uart io.ReadWriteCloser, conO *net.Conn, in []byte) {

	conn := *conO

	yellow := color.New(color.FgYellow).SprintFunc()
	// red := color.New(color.FgRed).SprintFunc()
	// green := color.New(color.FgGreen).SprintFunc()

	epta := color.New(color.FgBlue)
	epta.Printf("getResponse {%s}", string(in))

	// reO := regexp.MustCompile(`((?:.+|\n)+)\r\n`)
	res0 := strings.Split(string(in), "\r\n") //reO.FindAllString(string(in), -1)

	for _, item := range res0 {

		// fmt.Println("conn ", conn)

		if conn == nil {
			color.Red("conn is nil")
			continue
		}

		if len(item) == 0 {
			continue
		}

		color.Cyan(hex.Dump([]byte(item)))
		// item = strings.TrimRight(item, "\n")
		fmt.Printf(yellow("LAST: {%s}"), item)

		for _, vE := range voltageErrors {
			if strings.Index(item, vE) > -1 {
				color.Red("Under-voltage")
				fmt.Fprintf(conn, ("{\"command\":\"Under-voltage\"}\n"))
			}
		}
		if strings.Index(item, "+CMTE") > -1 {
			color.Red("Device is running HOT")
			fmt.Fprintf(conn, ("{\"command\":\"Device is running HOT\"}\n"))
		} else if strings.Index(item, "NO CARRIER") > -1 {
			color.Red("Call ended")
			fmt.Fprintf(conn, ("{\"command\":\"callEnd\"}\n"))
		} else if strings.Index(item, "MISSED_CALL:") > -1 {
			re := regexp.MustCompile(`MISSED\_CALL\:\s\d{2}\:\d{2}(?:AM|PM)\s(\+?\d+)`)
			res := re.FindStringSubmatch(item)

			if len(res) > 0 {
				color.Yellow("Missed Call %s", res[1])
				fmt.Fprintf(conn, ("{\"command\":\"missedCall\", \"phone\":\"%s\"}\n"), res[1])
			}
		} else if strings.Index(item, "+CUSD:") > -1 {
			getCUSD(item)
		} else if strings.Index(item, "+CLIP:") > -1 {

			// color.Yellow("CLIP!!!!!!!!!!")

			re := regexp.MustCompile(`\+CLIP\:\s+\"(\+\d+)"`)
			res := re.FindStringSubmatch(item)

			if len(res) > 0 {
				if res[1] != "" {
					color.Yellow("Phone number %s", res[1])
					fmt.Fprintf(conn, "{\"command\":\"incoming\", \"phone\":\"%s\"}\n", res[1])
				}
			}
		} else if strings.Index(item, "+CMTI:") > -1 {
			re := regexp.MustCompile(`\+CMTI\:\s+\"(?:\w+|\s+)+\",(\d+)`)
			res := re.FindStringSubmatch(item)

			if len(res) > -1 {
				color.Yellow("New SMS %s", res[1])
				// fmt.Fprintf(conn, yellow("{\"response\":\"new SMS\"}\n"))
				time.Sleep(time.Second)
				fmt.Fprintf(uart, "AT+CMGR=%s\r\n", res[1])
			}
		} else if strings.Index(item, "+CMT:") > -1 {
			getSMS(item)
		} else if strings.Index(item, "+CMGR:") > -1 {
			color.Red("Asd")
			lastCommand = append(lastCommand, item[:]...)
			item = string(lastCommand)

			re := regexp.MustCompile(`\+CMGR\:\s\d+,\"(?:.+)?\",\d+\r\n(.+|\n+)\r\n`)
			rssi := re.FindStringSubmatch(item)

			fmt.Printf("REGEX CMGR: %v\n", rssi)

			if len(rssi) > 0 {

				convBytes := []rune(rssi[1])

				index, _ := hex.DecodeString(string(convBytes[26*2 : 27*2]))
				typeI, _ := hex.DecodeString(string(convBytes[36:38]))

				color.Green(string(convBytes[26*2 : 27*2]))

				index2 := len(convBytes) - int(index[0])*2
				phone := parsePhone(convBytes[22:34])
				converted, _ := hex.DecodeString(string(convBytes[index2:]))

				text1 := DecodePDU7(string(convBytes[index2:]))

				text2 := string(UCS2.Decode(converted))

				color.Yellow("PDU: %s", rssi[1])
				color.Yellow("Length: %d", index) //1	1
				color.Yellow("phone: %s", phone)
				color.Yellow("phone: %s", phone)
				color.Yellow("SMS: %s", string(convBytes[index2:]))
				color.Yellow("type: %s", hex.Dump(typeI))
				color.Yellow("PDU7: %q", text1)
				color.Yellow("UCS2: %q", text2)

				// fmtet := fmt.Sprintf("%q", text1)
				out := ""
				// if []rune(fmtet)[1] == []rune("\\")[0] && []rune(fmtet)[2] == []rune("x")[0] {
				if typeI[0] == 0x08 {
					out = text2
				} else {
					out = text1
				}

				fmt.Fprintf(conn, ("{\"command\":\"recieveSMS\", \"phone\":\"%s\", \"text\": %q}\n"), phone, out)
			}

		} else if strings.Index(item, "+CSQ:") > -1 {
			re := regexp.MustCompile(`\+CSQ:\s+(\d+),`)
			rssi := re.FindStringSubmatch(item)

			if len(rssi) > 0 {
				color.Yellow("RSSI %s", rssi[1])
				fmt.Fprintf(conn, "{\"command\":\"rssi\",\"text\":\"%s\"}\n", rssi[1])
			}

		} else if strings.Index(item, "+CREG:") > -1 {
			re := regexp.MustCompile(`\+CREG:\s+\d+,(\d+)`)
			stat := re.FindStringSubmatch(item)

			fmt.Println(stat)

			if len(stat) > 0 {
				color.Yellow("The network is %s", (map[bool]string{true: "connected", false: "searching"})[stat[1] != "0"])
				fmt.Fprintf(conn, ("{\"command\":\"connected\", \"text\":\"%s\"}\n"), (map[bool]string{true: "true", false: "false"})[stat[1] != "0"])
			}

		} else if strings.Index(item, "BUSY") > -1 {
			fmt.Fprintf(conn, ("{\"command\":\"callEnd\"}\n"))
		} else if strings.Index(item, "ERROR") > -1 {
			color.Red("ERROR")
			fmt.Fprintf(conn, ("{\"command\":\"ERROR\"}\n"))
		} else if strings.Index(item, "OK") > -1 {
			color.Green("OK")
			// fmt.Fprintf(conn, green("{\"command\":\"OK\"}"))
		} else if strings.Index(item, "VOICE CALL: END: ") > -1 {
			var duration int
			_, err := fmt.Sscanf(item, "VOICE CALL: END: %06d", &duration)
			if err != nil {
				panic(err)
			} else {
				color.Yellow("Call Duration is %d", duration)
				fmt.Fprintf(conn, ("{\"command\":\"duration\",\"text\":\"%d\"}\n"), duration)
			}
		} else {
			lastCommand = append(lastCommand, item[:]...)
			color.Red("LAST response is undefined")

			continue
		}

		lastCommand = lastCommand[:0]

	}

	if len(in) == 1 && in[0] == 0 {
		color.Red("Restart?!")
		fmt.Fprintf(conn, ("{\"command\":\"Restart?!\"}\n"))
	}
	// return out
}

func recieve(con0 *net.Conn, s io.ReadWriteCloser) {
	conn := *con0
	for {
		time.Sleep(time.Second)
		color.Cyan("READING...")
		buf := make([]byte, 512)
		n, err := s.Read(buf)
		if err != nil {
			fmt.Print(err)
		}
		color.Magenta("RAW:")
		color.Magenta(hex.Dump(buf[:n]))
		lastCommand = append(lastCommand, buf[:n]...)
		// fmt.Printf("BEFORE{%s}\n", string(lastCommand))

		if len(buf[:n]) == 1 && buf[0] == 0x00 {
			red := color.New(color.FgRed).SprintFunc()
			fmt.Fprintf(conn, red("{\"warning\":\"voltage error\"}\n"))
		}

		if strings.HasSuffix(string(buf[:n]), "\n") || strings.HasSuffix(string(buf[:n]), string([]byte{0x0d, 0x0a, 0x00})) {
			color.Green("just \\n")
			getResponse(s, con0, lastCommand)
			lastCommand = lastCommand[:0]
			// fmt.Printf("After{%s}\n", string(lastCommand))
		}

	}
}

func formatPhone(in string) string {
	out := strings.Replace(in, " ", "", -1)
	out = strings.Replace(out, "-", "", -1)
	out = strings.Replace(out, "(", "", -1)
	out = strings.Replace(out, ")", "", -1)
	out = strings.Replace(out, "+", "", -1)

	if []rune(out)[0] == []rune("8")[0] {
		out = strings.TrimLeft(out, "8")
		out = "7" + out
	}

	if len(out) == 11 {
		out = "+" + out
		return out
	}

	return "ERROR"

}

//Encode is to encode
func (s UCS2) Encode() []byte {
	e := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
	es, _, err := transform.Bytes(e.NewEncoder(), s)
	if err != nil {
		return s
	}
	return es
}

// Decode from UCS2.
func (s UCS2) Decode() []byte {
	e := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
	es, _, err := transform.Bytes(e.NewDecoder(), s)
	if err != nil {
		return s
	}
	return es
}

func parsePhone(in []rune) string {
	var out [12]rune
	for i := 0; i < 6; i++ {
		out[2*i] = in[2*i+1]
		out[2*i+1] = in[2*i]
	}
	return string(out[:11])
}

func encodePhone(in string) string {
	raw := []rune(strings.TrimLeft(in, "+"))
	raw = append(raw, []rune("F")[0])

	var out [12]rune
	for i := 0; i < 6; i++ {
		out[2*i] = raw[2*i+1]
		out[2*i+1] = raw[2*i]
	}

	return string(out[:])
}

// DecodePDU7 is nice
func DecodePDU7(h string) (result string) {
	var binstr string

	b, err := hex.DecodeString(h)
	if err != nil {
		// May be you need to raise exception here
		return ""
	}
	for i := len(b) - 1; i >= 0; i-- {
		binstr += fmt.Sprintf("%08b", b[i])
	}
	for len(binstr) > 0 {
		p := int(math.Max(float64(len(binstr)-7), 0))
		b, _ := strconv.ParseUint(binstr[p:], 2, 8)
		result += string(b)
		binstr = binstr[:p]
	}

	return result
}

func externalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}

func tcpLoop(conn *net.Conn, uart io.ReadWriteCloser) {
	ln, _ := net.Listen("tcp", fmt.Sprintf(":%d", tcpPort))
	color.Blue("Waiting for TCP connection...")
	var err error
	for {
		*conn, err = ln.Accept()

		fmt.Println("conn ", conn)

		if err != nil {
			color.Red("Error TCP connection: ", err.Error())
			os.Exit(1)
		}
		time.Sleep(10 * time.Millisecond)
		// wg.Add(1)
		go checkTCP(conn, uart)
	}
}

func getCUSD(item string) {
	color.Green("getResponse+CUSD")
	re := regexp.MustCompile(`\+CUSD:\s\d+,(?:\s+)?\"((?:(?:.+)|\n+)+)\"?,\s?\d+`)
	res := re.FindStringSubmatch(item)

	fmt.Printf("REGEX CUSD: %v\n", res)

	if len(res) > 0 {

		converted := ""

		convBytes, epta := hex.DecodeString(res[1])
		if epta != nil {
			converted = res[1]
		} else {
			converted = string(UCS2.Decode(convBytes))
		}

		color.Yellow("The USSD result is %s", converted)
		fmt.Fprintf(conn, ("{\"command\":\"USSD\",\"text\":%q}\n"), converted)
	} else {
		// cusdFlag = true
		getCUSD(string(lastCommand) + item)
	}
}

func getSMS(item string) {
	color.Green("getSMS +CMT:")
	re := regexp.MustCompile(`\+CMT:\s\"\",\d+(?:\r|\n|\r\n)([A-F0-9a-f]+)`)
	res := re.FindStringSubmatch(item)

	fmt.Printf("REGEX SMS: %v\n", res)

	if len(res) > 0 {
		convBytes := []rune(res[1])

		index, _ := hex.DecodeString(string(convBytes[26*2 : 27*2]))
		typeI, _ := hex.DecodeString(string(convBytes[36:38]))

		color.Green(string(convBytes[26*2 : 27*2]))

		var index2 int

		if typeI[0] == 0x08 {
			index2 = len(convBytes) - int(index[0])*2
		} else {
			index2 = len(convBytes) - LenghtInSep(int(index[0]))*2
		}

		phone := parsePhone(convBytes[22:34])
		converted, _ := hex.DecodeString(string(convBytes[index2:]))

		text1 := DecodePDU7(string(convBytes[index2:]))

		text2 := string(UCS2.Decode(converted))

		color.Yellow("PDU: %s", res[1])
		color.Yellow("Length: %d", index) //1	1
		color.Yellow("phone: %s", phone)
		// color.Yellow("phone: %s", phone)
		color.Yellow("SMS: %s", string(convBytes[index2:]))
		color.Yellow("type: %s", hex.Dump(typeI))
		color.Yellow("PDU7: %q", text1)
		color.Yellow("UCS2: %q", text2)

		// fmtet := fmt.Sprintf("%q", text1)
		out := ""
		// if []rune(fmtet)[1] == []rune("\\")[0] && []rune(fmtet)[2] == []rune("x")[0] {
		if typeI[0] == 0x08 {
			out = text2
			color.Red("Hextets: %d", index) //1	1
			color.Yellow("UCS2: %q", text2)
		} else {
			out = text1
			color.Red("Septets: %d", LenghtInSep(int(index[0]))) //1	1
			color.Yellow("PDU7: %q", text1)
		}

		fmt.Fprintf(conn, ("{\"command\":\"recieveSMS\", \"phone\":\"%s\", \"text\": %q}\n"), phone, strings.TrimRight(out, "\x00"))
	} else {
		// cusdFlag = true
		getSMS(string(lastCommand) + item)
	}
}

func LenghtInSep(in int) int {
	var out float64
	out = float64(in) * 7.0 / 8.0
	chk := math.Floor(out)
	if chk == out {
		return int(out)
	}
	return int(chk) + 1
}
