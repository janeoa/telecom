package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gordonklaus/portaudio"
)

const sampleRate = 44100
const seconds = 0.01

func main() {
	portaudio.Initialize()
	defer portaudio.Terminate()
	buffer := make([]float32, sampleRate*seconds)

	p := make([]byte, sampleRate*seconds*4)

	// fmt.Print("the p is ")
	// fmt.Println(len(p))
	// time.Sleep(time.Second * 2)

	addr := net.UDPAddr{
		Port: 1234,
		IP:   net.ParseIP("192.168.25.30"),
	}
	ser, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Printf("Some error %v\n", err)
		return
	}

	stream, err := portaudio.OpenDefaultStream(0, 1, sampleRate, len(buffer), func(out []float32) {

		_, _, err := ser.ReadFromUDP(p)
		chk(err)

		buffOut := buffToFloatArr(p)

		for i := range out {
			out[i] = buffOut[i]
		}
	})
	chk(err)
	chk(stream.Start())
	time.Sleep(time.Second * 40)
	chk(stream.Stop())
	defer stream.Close()

	if err != nil {
		fmt.Println(err)
	}

	ctlc := make(chan os.Signal)
	signal.Notify(ctlc, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
	<-ctlc

}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

func Float32frombytes(bytes []byte) float32 {
	bits := binary.LittleEndian.Uint32(bytes)
	float := math.Float32frombits(bits)
	return float
}

func buffToFloatArr(buff []byte) []float32 {
	var result [sampleRate * seconds]float32
	for i := 0; i < len(buff)/4; i++ {
		// fmt.Println(buff[i:i+4])
		// fmt.Println(Float32frombytes(buff[i:i+4]))
		result[i] = Float32frombytes(buff[i*4 : i*4+4])
	}
	return result[:]
}
