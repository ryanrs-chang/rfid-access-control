package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/caarlos0/env"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/jacobsa/go-serial/serial"
)

func display(buf []byte) {
	for i := 0; i < len(buf); i++ {
		fmt.Printf("%x ", buf[i])
	}
	fmt.Printf("\n")
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
		return -1, -1, errors.New(fmt.Sprintf("format error: length or head or last byte: buf:%x, len:%d head:%x, last:%x", buf, len(buf), buf[0], buf[len(buf)-1]))
	}

	checksum := transfor2HexAndCalsChecksum(buf[1:15])
	if checksum != buf[15] || ^checksum != buf[16] {
		return -1, -1, errors.New("checksum error")
	}

	country := reverse2int(buf[11:15])
	id := reverse2int(buf[1:11])

	return country, id, nil
}

type config struct {
	Port     string `env:"PORT" envDefault:"/dev/ttyACM0"`
	BaudRate uint   `env:"BaudRate" envDefault:"9600"`
	DataBits uint   `env:"DataBits" envDefault:"8"`
	StopBits uint   `env:"StopBits" envDefault:"1"`
	MQTTHost string `env:"MQTTHost" envDefault:"tcp://iot.eclipse.org:1883"`
}

var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	log.Printf("TOPIC: %s MSG:%s\n", msg.Topic(), msg.Payload())
}

func main() {
	cfg := config{}
	err := env.Parse(&cfg)
	if err != nil {
		log.Printf("%+v\n", err)
	}

	options := serial.OpenOptions{
		PortName:        cfg.Port,
		BaudRate:        cfg.BaudRate,
		DataBits:        cfg.DataBits,
		StopBits:        cfg.StopBits,
		MinimumReadSize: 18,
	}

	log.Printf("Comport: %s %d %d %d", options.PortName, options.BaudRate, options.DataBits, options.StopBits)

	// Open the port.
	port, err := serial.Open(options)
	if err != nil {
		log.Fatalf("serial.Open: %v", err)
	}
	reader := bufio.NewReader(port)
	defer port.Close()

	opts := MQTT.NewClientOptions().AddBroker(cfg.MQTTHost)
	opts.SetClientID("rfid_recoder")
	opts.SetDefaultPublishHandler(f)

	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	if token := c.Subscribe("pool/abc/rfid", 0, nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	for {
		buf, err := reader.ReadSlice(0x03)
		if err != nil {
			log.Printf("%v\n", err)
			continue
		}

		country, id, err := handleRFID(buf)
		if err != nil {
			log.Printf("%s\n", err)
			continue
		}

		log.Printf("%d%d\n", country, id)

		text := fmt.Sprintf("%d%d", country, id)
		token := c.Publish("pool/abc/rfid", 0, false, text)
		token.Wait()
	}
}
