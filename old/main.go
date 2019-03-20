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
const myIP = "192.168.25.30"
const connport = 1234
const secondIP = "192.168.25.31"

func main() {

	/** INIT */
	portaudio.Initialize()
	defer portaudio.Terminate()

	/** THE UPSTREAM */

	upconn, err := net.Dial("udp", fmt.Sprintf("%s:%d", secondIP, connport))

	upbuffer := make([]float32, sampleRate*seconds)
	upstream, err := portaudio.OpenDefaultStream(1, 0, sampleRate, len(upbuffer), func(in []float32) {
		for i := range upbuffer {
			upbuffer[i] = in[i]
		}

		upconn.Write(floatArrToByteArr(upbuffer))

		if err != nil {
			fmt.Printf("Some error %v", err)
			return
		}
	})
	chk(err)

	/** THE DOWNSTREAM */

	downbuffer := make([]float32, sampleRate*seconds)
	p := make([]byte, sampleRate*seconds*4)
	addr := net.UDPAddr{
		Port: connport,
		IP:   net.ParseIP(myIP),
	}
	ser, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Printf("Some error %v\n", err)
		return
	}
	downstream, err := portaudio.OpenDefaultStream(0, 1, sampleRate, len(downbuffer), func(out []float32) {

		_, _, err := ser.ReadFromUDP(p)
		chk(err)

		buffOut := buffToFloatArr(p)

		for i := range out {
			out[i] = buffOut[i]
		}
	})

	chk(err)
	chk(upstream.Start())
	chk(downstream.Start())

	time.Sleep(time.Second * 60 * 30)

	defer upstream.Close()
	defer upconn.Close()
	chk(downstream.Stop())
	defer downstream.Close()

	// ctlc := make(chan os.Signal)
	// signal.Notify(ctlc, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
	// <-ctlc
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

func floatArrToByteArr(f []float32) []byte {
	var result [sampleRate * seconds * 4]byte

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
