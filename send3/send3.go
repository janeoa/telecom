package main

import (
	"fmt"
	"net"
	"time"

	"github.com/gordonklaus/portaudio"
)

const sampleRate = 44100
const seconds = 0.01

func main() {

	// conn, err := net.Dial("udp", "192.168.25.31:1234")
	conn, err := net.Dial("udp", "192.168.0.103:2000")
	// fmt.Fprintf(conn, "is it up?!")
	// fmt.Fprintf(conn, "asdasd up?!")

	portaudio.Initialize()
	defer portaudio.Terminate()
	buffer := make([]int16, sampleRate*seconds)
	stream, err := portaudio.OpenDefaultStream(1, 0, sampleRate, len(buffer), func(in []int16) {
		for i := range buffer {
			buffer[i] = in[i] //fmt.Sprintf("%f", in[i])
		}

		conn.Write(int16ArrToByteArr(buffer))

		if err != nil {
			fmt.Printf("Some error %v", err)
			return
		}
	})

	// stream2, err := portaudio.OpenDefaultStream(0, 1, sampleRate, len(buffer), func(out []int16) {

	// 	for i := range out {
	// 		out[i] = buffer[i]
	// 	}
	// })
	// chk(stream2.Start())
	// defer stream2.Close()

	chk(err)
	chk(stream.Start())

	defer stream.Close()

	defer conn.Close()

	time.Sleep(time.Second * 60 * 30)
	// ctlc := make(chan os.Signal)
	// signal.Notify(ctlc, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
	// <-ctlc
}

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

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

// func floatArrToByteArr(f []float32) []byte {
// 	var result [sampleRate * seconds * 4]byte

// 	// fmt.Println(len(f))
// 	for i := 0; i < len(f); i++ {
// 		part := Float32bytes(f[i])
// 		for j := 0; j < 4; j++ {
// 			result[i*4+j] = part[j]
// 		}
// 	}

// 	return result[:]
// }

// func Float32bytes(float float32) []byte {
// 	bits := math.Float32bits(float)
// 	bytes := make([]byte, 4)
// 	binary.LittleEndian.PutUint32(bytes, bits)
// 	return bytes
// }
