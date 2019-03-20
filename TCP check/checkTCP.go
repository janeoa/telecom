package main

import (
	"fmt"
	"net"
)

func main() {

	p := make([]byte, 44100*0.02*2)

	// fmt.Print("the p is ")
	// fmt.Println(len(p))
	// time.Sleep(time.Second * 2)

	addr := net.UDPAddr{
		Port: 2000,
		IP:   net.ParseIP("192.168.25.18"),
	}
	ser, err := net.ListenUDP("udp", &addr)

	fmt.Printf("WUT?")

	for {
		_, _, err = ser.ReadFromUDP(p)
		fmt.Println("__nop__")
		fmt.Println(p)
		chk(err)

		if err != nil {
			fmt.Println(err)
		}
	}
}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}
