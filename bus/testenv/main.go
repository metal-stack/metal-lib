package main

import (
	"fmt"
	"github.com/metal-stack/metal-lib/bus"
	"github.com/nsqio/go-nsq"
	"log"
	"sync"
)

const (
	producerAddress = "metal-control-plane-nsqd:4150"
	consumerAddress = "localhost:4161"
	caCertFile      = "ca_cert.pem"
	clientCertFile  = "client.pem"
	topic           = "test"
)

func main() {
	cfg := bus.CreateNSQConfig(&bus.TLSConfig{
		CACertFile:     caCertFile,
		ClientCertFile: clientCertFile,
	})
	produce(cfg)
	consume(cfg)
}

func produce(cfg *nsq.Config) {
	fmt.Println("--- Producing...")

	p, err := nsq.NewProducer(producerAddress, cfg)
	if err != nil {
		log.Panic(err)
	}
	err = p.Publish(topic, []byte(`{"msg": "blubber"}`))
	if err != nil {
		log.Panic(err)
	}

	fmt.Println("OK")
}

func consume(cfg *nsq.Config) {
	fmt.Println("--- Consuming...")

	result := "failed"
	var wg sync.WaitGroup
	wg.Add(1)

	c, err := nsq.NewConsumer(topic, "chan", cfg)
	if err != nil {
		log.Panic(err)
	}
	c.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
		log.Printf("Received message: %s", string(message.Body))
		result = "succeeded"
		c.Stop()
		wg.Done()
		return nil
	}))
	err = c.ConnectToNSQLookupd(consumerAddress)
	if err != nil {
		log.Panic(err)
	}

	wg.Wait()

	fmt.Printf("---\nTest %s\n---\n", result)
}
