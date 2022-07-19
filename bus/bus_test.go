package bus

import (
	"os"
	"testing"

	"github.com/nsqio/nsq/nsqd"
	"go.uber.org/zap/zaptest"
)

var (
	tcpAddress  = "localhost:44150"
	httpAddress = "localhost:44151"
	publisher   Publisher
	consumer    *Consumer
)

func startupNSQD() error {
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
	consumer, err = NewConsumer(zaptest.NewLogger(&testing.T{}), nil)
	if err != nil {
		panic(err)
	}
	consumer.With(NSQDs(tcpAddress))

	cfg := &PublisherConfig{
		TCPAddress:   tcpAddress,
		HTTPEndpoint: httpAddress,
	}

	publisher, err = NewPublisher(zaptest.NewLogger(&testing.T{}), cfg)
	if err != nil {
		panic(err)
	}

	return nil
}

func TestMain(m *testing.M) {
	_ = startupNSQD()
	code := m.Run()
	os.Exit(code)
}
