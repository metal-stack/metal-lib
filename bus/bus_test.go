package bus

import (
	"log/slog"
	"os"
	"testing"

	"github.com/nsqio/nsq/nsqd"
)

var (
	tcpAddress  = "localhost:44150"
	httpAddress = "localhost:44151"
	publisher   Publisher
	consumer    *Consumer
)

func startupNSQD() {
	_, disable := os.LookupEnv("NO_NSQD_START")
	if !disable {
		opts := nsqd.NewOptions()
		opts.TCPAddress = tcpAddress
		opts.HTTPAddress = httpAddress
		opts.DataPath = "/tmp/"
		nsqd, err := nsqd.New(opts)
		if err != nil {
			panic(err)
		}
		go func() {
			err = nsqd.Main()
			if err != nil {
				panic(err)
			}
		}()
	}
	var err error
	consumer, err = NewConsumer(slog.Default(), nil)
	if err != nil {
		panic(err)
	}
	consumer.With(NSQDs(tcpAddress))

	cfg := &PublisherConfig{
		TCPAddress:   tcpAddress,
		HTTPEndpoint: httpAddress,
	}

	publisher, err = NewPublisher(slog.Default(), cfg)
	if err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	startupNSQD()
	code := m.Run()
	os.Exit(code)
}
