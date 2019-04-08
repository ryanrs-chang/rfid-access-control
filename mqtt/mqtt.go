package mqtt

import (
	"log"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	log.Printf("TOPIC: %s MSG:%s\n", msg.Topic(), msg.Payload())
}

func NewClient(host string, clinetID string) MQTT.Client {
	opts := MQTT.NewClientOptions().AddBroker(host)

	opts.SetClientID(clinetID)
	opts.SetDefaultPublishHandler(f)

	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	return c
}
