package bus

import (
	"os"
	"testing"

	"github.com/metal-stack/metal-lib/zapup"
	"github.com/nsqio/nsq/nsqd"
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
			return err
		}
		go func() {
			err = nsqd.Main()
			if err != nil {
				panic(err)
			}
		}()
	}
	var err error
	consumer, err = NewConsumer(zapup.MustRootLogger(), nil)
	if err != nil {
		return err
	}
	consumer.With(NSQDs(tcpAddress))

	cfg := &PublisherConfig{
		TCPAddress:   tcpAddress,
		HTTPEndpoint: httpAddress,
	}

	publisher, err = NewPublisher(zapup.MustRootLogger(), cfg)
	if err != nil {
		return err
	}

	return nil
}

func TestMain(m *testing.M) {
	_ = startupNSQD()
	code := m.Run()
	os.Exit(code)
}
