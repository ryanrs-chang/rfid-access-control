package device

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/jacobsa/go-serial/serial"
)

type Device struct {
	options serial.OpenOptions
	reader  *bufio.Reader
	io      io.ReadWriteCloser
}

func transfor2HexAndCalsChecksum(buf []byte) (checksum byte) {
	checksum = 0x00
	for i, len := 0, len(buf); i < len; i++ {
		checksum ^= buf[i]

		if buf[i] >= '0' && buf[i] <= '9' {
			buf[i] = buf[i] - '0'
		} else if buf[i] >= 'A' && buf[i] <= 'F' {
			buf[i] = buf[i] - 0x37
		}
	}
	return checksum
}

func reverse2int(buf []byte) int {
	ret := 0
	for i := len(buf) - 1; i >= 0; i-- {
		ret = ret ^ int(buf[i])<<uint(i*4)
	}
	return ret
}

func handleRFID(buf []byte) (int, int, error) {
	if len(buf) != 18 || buf[0] != 0x02 || buf[17] != 0x03 {
		return -1, -1, fmt.Errorf("format error: length or head or last byte: buf:%x, len:%d head:%x, last:%x", buf, len(buf), buf[0], buf[len(buf)-1])
	}

	checksum := transfor2HexAndCalsChecksum(buf[1:15])
	if checksum != buf[15] || ^checksum != buf[16] {
		return -1, -1, errors.New("checksum error")
	}

	country := reverse2int(buf[11:15])
	id := reverse2int(buf[1:11])

	return country, id, nil
}

// Open open Serial Port
func (d *Device) Open(options serial.OpenOptions) {
	// Open the port.
	io, err := serial.Open(options)
	if err != nil {
		log.Fatalf("Open failed: %v", err)
	}
	d.reader = bufio.NewReader(io)
}

// Listen wating data writing to buffer
func (d *Device) Listen(cb func(int, int)) {
	log.Println("listening..")
	for {
		buf, err := d.reader.ReadSlice(0x03)
		if err != nil {
			errText := fmt.Sprintf("%s", err)
			if errText == "multiple Read calls return no data or error" {
				time.Sleep(1000)
				continue
			}
			log.Panicln(err)
		}

		country, id, err := handleRFID(buf)
		if err != nil {
			log.Printf("%s\n", err)
			continue
		}

		if cb != nil {
			cb(country, id)
		}

		time.Sleep(50)
	}
}

// Close close serial port
func (d *Device) Close() error {
	return d.io.Close()
}

// NewDevice new instance for Device
func NewDevice() *Device {
	return &Device{}
}
