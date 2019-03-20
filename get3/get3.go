package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/gordonklaus/portaudio"
)

const sampleRate = 8000
const seconds = 0.01

func main() {

	// ept2 := []int16{-1890, 510}
	// epta := int16ArrToByteArr(ept2) //[]byte{0xe3, 0x4a, 0xe3, 0x4a}
	// fmt.Printf("%x %x %x %x", epta[0], epta[1], epta[2], epta[3])
	// fmt.Println(bytesToInt16Arr(epta))

	portaudio.Initialize()
	defer portaudio.Terminate()
	buffer := make([]int16, sampleRate*seconds)

	p := make([]byte, sampleRate*seconds*2)

	// fmt.Print("the p is ")
	// fmt.Println(len(p))
	// time.Sleep(time.Second * 2)

	addr := net.UDPAddr{
		Port: 2000,
		IP:   net.ParseIP("192.168.0.100"),
	}
	ser, err := net.ListenUDP("udp", &addr)

	fmt.Printf("WUT?")

	if err != nil {
		fmt.Printf("Some error %v\n", err)
		return
	}

	fmt.Printf("IDK")

	stream, err := portaudio.OpenDefaultStream(0, 1, sampleRate, len(buffer), func(out []int16) {

		_, _, err := ser.ReadFromUDP(p)
		// fmt.Print("__nop__")
		// fmt.Println(p)
		chk(err)

		buffOut := bytesToInt16Arr(p)

		for i := range out {
			out[i] = buffOut[i]
		}
	})
	chk(err)
	chk(stream.Start())
	time.Sleep(time.Second * 60 * 30)
	chk(stream.Stop())
	defer stream.Close()

	if err != nil {
		fmt.Println(err)
	}

	// ctlc := make(chan os.Signal)
	// signal.Notify(ctlc, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
	// <-ctlc

}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

func bytesToInt16Arr(in []byte) []int16 {
	var out [sampleRate * seconds]int16
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

// func Float32frombytes(bytes []byte) float32 {
// 	bits := binary.LittleEndian.Uint32(bytes)
// 	float := math.Float32frombits(bits)
// 	return float
// }

// func buffToFloatArr(buff []byte) []float32 {
// 	var result [sampleRate * seconds]float32
// 	for i := 0; i < len(buff)/4; i++ {
// 		// fmt.Println(buff[i:i+4])
// 		// fmt.Println(Float32frombytes(buff[i:i+4]))
// 		result[i] = Float32frombytes(buff[i*4 : i*4+4])
// 	}
// 	return result[:]
// }

func int16ArrToByteArr(in []int16) []byte {
	var out [sampleRate * seconds * 2]byte
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
