package mqtt

import (
	"crypto/tls"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	uuid "github.com/satori/go.uuid"
)

type Config struct {
	Host     string
	ClientID string
	Username string
	Password string
}

var f mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Printf("TOPIC: %s MSG:%s\n", msg.Topic(), msg.Payload())
}

func NewClient(cfg *Config, tlsCfg *tls.Config) mqtt.Client {
	clinetID := cfg.ClientID
	if clinetID == "" {
		clinetID = uuid.NewV4().String()
	}

	opts := mqtt.NewClientOptions().
		AddBroker(cfg.Host).
		SetDefaultPublishHandler(f).
		SetClientID(clinetID).
		SetUsername(cfg.Username).
		SetPassword(cfg.Password)

	if tlsCfg != nil {
		opts.SetTLSConfig(tlsCfg)
	}

	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	return c
}
