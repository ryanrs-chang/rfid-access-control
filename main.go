package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"rfid-access-control/device"
	"rfid-access-control/mqtt"

	"github.com/caarlos0/env"
	"github.com/jacobsa/go-serial/serial"
)

type Config struct {
	// for rfid reader
	Port     string `env:"PORT" envDefault:"/dev/ttyACM0"`
	BaudRate uint   `env:"BaudRate" envDefault:"9600"`
	DataBits uint   `env:"DataBits" envDefault:"8"`
	StopBits uint   `env:"StopBits" envDefault:"1"`
	// for mqtt
	Host     string `env:"Host" envDefault:"tcp://iot.eclipse.org:1883"`
	ClientID string `env:"ClientID" envDefault:"client"`
	Topic    string `env:"Topic" envDefault:"pool/abc/rfid"`
	CertFile string `env:"CertFile"`
	KeyFile  string `env:"KeyFile"`
	Username string `env:"Username"`
	Password string `env:"Password"`
	// logger
	LogFile string `env:"logFile" envDefault:"reader.log"`
}

func getInsecureCert(cfg Config) *tls.Config {
	if cfg.CertFile == "" && cfg.KeyFile == "" {
		return nil
	}

	cer, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	conf := &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cer},
	}

	return conf
}

func main() {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		fmt.Printf("%+v\n", err)
	}
	fmt.Printf("%+v\n", cfg)

	options := serial.OpenOptions{
		PortName:        cfg.Port,
		BaudRate:        cfg.BaudRate,
		DataBits:        cfg.DataBits,
		StopBits:        cfg.StopBits,
		MinimumReadSize: 18,
	}

	log.Printf("Serial Port: %s %d %d %d", options.PortName, options.BaudRate, options.DataBits, options.StopBits)

	// for rfid device
	device := device.NewDevice()
	device.Open(options)
	defer device.Close()

	// for mqtt client
	tlsCfg := getInsecureCert(cfg)

	clientCfg := &mqtt.Config{
		Host:     cfg.Host,
		ClientID: cfg.ClientID,
		Username: cfg.Username,
		Password: cfg.Password,
	}
	client := mqtt.NewClient(clientCfg, tlsCfg)
	if token := client.Subscribe(cfg.Topic, 0, nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	// for logger
	file, err := os.Create(cfg.LogFile)
	defer file.Close()
	if err != nil {
		log.Fatalln(err)
	}
	infoLog := log.New(file, "[ID] ", log.LstdFlags)

	device.Listen(func(country int, id int) {
		log.Printf("Country:%d, ID:%d", country, id)
		text := fmt.Sprintf("%d%d", country, id)
		infoLog.Println(text)

		token := client.Publish(cfg.Topic, 0, false, text)
		token.Wait()
	})
}
