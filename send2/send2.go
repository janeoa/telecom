package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"time"

	"github.com/gordonklaus/portaudio"
)

const sampleRate = 44100
const seconds = 0.01

func main() {

	// conn, err := net.Dial("udp", "192.168.25.31:1234")
	conn, err := net.Dial("udp", "192.168.25.31:1234")
	// fmt.Fprintf(conn, "is it up?!")
	// fmt.Fprintf(conn, "asdasd up?!")

	portaudio.Initialize()
	defer portaudio.Terminate()
	buffer := make([]float32, sampleRate*seconds)
	stream, err := portaudio.OpenDefaultStream(1, 0, sampleRate, len(buffer), func(in []float32) {
		for i := range buffer {
			buffer[i] = in[i] //fmt.Sprintf("%f", in[i])
		}

		conn.Write(floatArrToByteArr(buffer))

		if err != nil {
			fmt.Printf("Some error %v", err)
			return
		}
	})
	chk(err)
	chk(stream.Start())
	defer stream.Close()
	defer conn.Close()

	time.Sleep(time.Second * 40)
	// ctlc := make(chan os.Signal)
	// signal.Notify(ctlc, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
	// <-ctlc
}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

func floatArrToByteArr(f []float32) []byte {
	var result [sampleRate * seconds * 4]byte

	// fmt.Println(len(f))
	for i := 0; i < len(f); i++ {
		part := Float32bytes(f[i])
		for j := 0; j < 4; j++ {
			result[i*4+j] = part[j]
		}
	}

	return result[:]
}

func Float32bytes(float float32) []byte {
	bits := math.Float32bits(float)
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, bits)
	return bytes
}
