package main

import (
	"fmt"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

var in1 = []byte{0x00, 0x37, 0x00, 0x36, 0x00, 0x31, 0x00, 0x2E, 0x00, 0x32, 0x00, 0x33, 0x04, 0x40, 0x00, 0x2E, 0xd8, 0x3d, 0xdd, 0x13}

func main() {
	fmt.Println(string(UCS2.Decode(in1)))
}

// UCS2 text codec.
type UCS2 []byte

// Encode to UCS2.
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
