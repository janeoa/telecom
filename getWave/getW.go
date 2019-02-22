package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	p := make([]byte, 2048)
	conn, err := net.Dial("udp", "localhost:1234")
	if err != nil {
		fmt.Printf("Some error %v", err)
		return
	}
	fmt.Fprintf(conn, "Hi UDP Server, How are you doing?")
	_, err = bufio.NewReader(conn).Read(p)
	if err == nil {
		f, err := os.Create("test.txt")
		if err != nil {
			fmt.Println(err)
			return
		}
		l, err := f.Write(p)
		if err != nil {
			fmt.Println(err)
			f.Close()
			return
		}
		fmt.Println(l, "bytes written successfully")
		err = f.Close()
		if err != nil {
			fmt.Println(err)
			return
		}
		// fmt.Printf("%s\n", p)
	} else {
		fmt.Printf("Some error %v\n", err)
	}
	conn.Close()
}
