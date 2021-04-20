package bus

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/metal-stack/metal-lib/zapup"
	"github.com/nsqio/go-nsq"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// tests if the handler returns prematurely with specific error if a timeout occurs
func TestTimeoutWrapper_FailTimeout(t *testing.T) {

	e := &nsq.Message{
		Body: []byte("{}"),
	}

	twTimeout := timeoutWrapper{
		timeout: 100 * time.Millisecond,
		msgType: reflect.TypeOf(e),
		recv: func(i interface{}) error {
			time.Sleep(150 * time.Millisecond)
			return nil
		},
	}

	err := twTimeout.handleWithTimeout(e)
	if err != nil {
		t.Fatalf("no error expected, timeout error should be handled in onTimeout-function")
	}
}

// tests if the handler returns prematurely with call of the onTimeout if a timeout occurs, onTimeout returns no error
func TestTimeoutWrapper_FailTimeoutWithTimeoutFunc(t *testing.T) {

	messageFromQueue := &nsq.Message{
		Body: []byte("{}"),
	}

	// record if the timeoutfunc is called
	handlerCalled := false

	twTimeout := timeoutWrapper{
		timeout: 100 * time.Millisecond,
		onTimeout: func(err TimeoutError) error {
			handlerCalled = true

			// check that timeout is reported

			if err.Event() == nil {
				t.Errorf("timeout event expected")
			}

			return nil
		},
		msgType: reflect.TypeOf(messageFromQueue),
		recv: func(i interface{}) error {
			time.Sleep(150 * time.Millisecond)
			return nil
		},
	}

	err := twTimeout.handleWithTimeout(messageFromQueue)
	if err != nil {
		t.Errorf("no error expected because the onTimeout retuns nil")
	}
	if !handlerCalled {
		t.Errorf("timeoutfunction expected ")
	}
}

// tests if the handler returns prematurely with specific error if a timeout occurs
func TestTimeoutWrapper_OK_NoTimeout(t *testing.T) {

	e := &nsq.Message{
		Body: []byte("{}"),
	}

	twTimeout := timeoutWrapper{
		timeout: 0,
		msgType: reflect.TypeOf(e),
		recv: func(i interface{}) error {
			time.Sleep(500 * time.Millisecond)
			return nil
		},
	}

	err := twTimeout.handleWithTimeout(e)
	if err != nil {
		t.Fatalf("expected that handler returns without timeout")
	}
}

// tests the ok case, in which the handler finishes execution within the given timeout
func TestTimeoutWrapper_OK(t *testing.T) {

	e := &nsq.Message{
		Body: []byte("{}"),
	}

	twTimeout := timeoutWrapper{
		timeout: 50 * time.Millisecond,
		msgType: reflect.TypeOf(e),
		recv: func(i interface{}) error {
			time.Sleep(30 * time.Millisecond)
			return nil
		},
	}

	err := twTimeout.handleWithTimeout(e)
	if err != nil {
		t.Fatalf("expected that handler returns without timeout")
	}
}

// tests the ok case, in which the message age is in the valid range
func TestTimeoutWrapper_TTL_OK(t *testing.T) {

	e := &nsq.Message{
		Timestamp: time.Now().UnixNano(),
		Body:      []byte("{}"),
	}

	// this will be set in receive
	result := ""

	twTimeout := timeoutWrapper{
		ttl:     1 * time.Second,
		msgType: reflect.TypeOf(e),
		recv: func(i interface{}) error {
			result = "ok"
			return nil
		},
	}

	err := twTimeout.handleWithTimeout(e)
	if err != nil {
		t.Fatalf("expected that handler returns without timeout")
	}
	require.Equal(t, "ok", result)
}

// tests the ok case, in which the handler finishes execution within the given timeout
func TestTimeoutWrapper_TTL_DropMessage(t *testing.T) {

	e := &nsq.Message{
		Timestamp: time.Now().UnixNano(),
		Body:      []byte("{}"),
	}

	twTimeout := timeoutWrapper{
		ttl:     100 * time.Millisecond,
		msgType: reflect.TypeOf(e),
		recv: func(i interface{}) error {

			// message must be dropped
			t.Fatal("message must be dropped but was received!")
			return nil
		},
	}

	// wait to exceed ttl
	time.Sleep(200 * time.Millisecond)

	err := twTimeout.handleWithTimeout(e)
	if err != nil {
		t.Fatalf("expected that handler returns without timeout")
	}
}

type Msg struct {
	Name string
	Num  int
}

func TestNewPublisher(t *testing.T) {

	err := publisher.CreateTopic("topic42")

	var netErr net.Error
	if errors.As(err, &netErr) {
		// network error, no nsq running, skip roundtrip tests
		t.SkipNow()
	}

	if err != nil {
		t.Errorf("unexpected error, %v", err)
	}

	msg := Msg{
		Name: "mymsg",
		Num:  42,
	}
	err = publisher.Publish("topic42", msg)
	if err != nil {
		t.Error(err)
	}

	ch := make(chan int)

	err = consumer.MustRegister("topic42", "node42").Consume(Msg{}, func(m interface{}) error {
		fmt.Printf("received %v\n", m)
		ch <- 1
		return nil
	}, 1)
	if err != nil {
		t.Error(err)
	}

	timeout := time.After(5 * time.Second)
	select {
	case <-ch:
		// ok
	case <-timeout:
		t.Errorf("timeout")
	}
}

func TestNewConsumer(t *testing.T) {
	c, err := NewConsumer(zapup.MustRootLogger(), nil)
	if err != nil {
		t.Error(err)
	}
	err = c.MustRegister("topic", "channel").
		Consume(Msg{}, func(i interface{}) error {
			return nil
		}, 1)

	if err != nil {
		t.Errorf("unexpected error, %v", err)
	}
}

func TestNewConsumerLogLevel(t *testing.T) {
	c, err := NewConsumer(zapup.MustRootLogger(), nil)
	if err != nil {
		t.Error(err)
	}
	err = c.With(LogLevel(Debug)).
		MustRegister("topic", "channel").
		Consume(Msg{}, func(i interface{}) error {
			return nil
		}, 1)

	if err != nil {
		t.Errorf("unexpected error, %v", err)
	}
}

func TestNewConsumerWithTimeout(t *testing.T) {
	c, err := NewConsumer(zapup.MustRootLogger(), nil)
	if err != nil {
		t.Error(err)
	}
	err = c.With(LogLevel(Debug)).
		MustRegister("topic", "channel").
		Consume(Msg{}, func(i interface{}) error {
			// receiver, event handler
			return nil
		}, 1,
			Timeout(30*time.Second, func(err TimeoutError) error {
				// timeout handler
				return nil
			}))

	if err != nil {
		t.Errorf("unexpected error, %v", err)
	}
}

func TestNewConsumer_MultipleConsumeError(t *testing.T) {
	c, err := NewConsumer(zapup.MustRootLogger(), nil)
	if err != nil {
		t.Error(err)
	}
	cr := c.With(LogLevel(Debug)).
		MustRegister("topic", "channel")

	err = cr.Consume(Msg{}, func(i interface{}) error {
		// receiver, event handler
		return nil
	}, 1,
		Timeout(30*time.Second, func(err TimeoutError) error {
			// timeout handler
			return nil
		}))

	if err != nil {
		t.Errorf("unexpected error, %v", err)
	}

	// second

	err = cr.Consume(Msg{}, func(i interface{}) error {
		// receiver, event handler
		return nil
	}, 1,
		Timeout(30*time.Second, func(err TimeoutError) error {
			// timeout handler
			return nil
		}))

	if err == nil || err.Error() != "already connected" {
		t.Errorf("expected error, already connected")
	}
}

func TestBridgeNsqLogToCoreLog(t *testing.T) {

	type Test struct {
		NsqMsg string
		Level  zapcore.Level
	}

	tests := []Test{
		{
			NsqMsg: "INF  123 [switch/fra-equ01-leaf01] querying nsqlookupd http://metal.test.metal-stack.io:4161/lookup?topic=switch {\"app\": \"metal-core\"} ",
			Level:  zap.InfoLevel,
		},
		{
			NsqMsg: "ERR    2 [switch/fra-equ01-leaf01] error querying nsqlookupd (http://metal.test.metal-stack.io:4161/lookup?topic=switch) - Get http://metal.test.metal-stack.io:4161/lookup?topic=switch: dial tcp: i/o timeout        {\"app\": \"metal-core\"}",
			Level:  zap.ErrorLevel,
		},
		{
			NsqMsg: "WRN    1 [switch/fra-equ01-leaf01] error querying nsqlookupd (http://metal.test.metal-stack.io:4161/lookup?topic=switch) - Get http://metal.test.metal-stack.io:4161/lookup?topic=switch: dial tcp: i/o timeout        {\"app\": \"metal-core\"}",
			Level:  zap.WarnLevel,
		},
		{
			NsqMsg: "DBG    1 [switch/fra-equ01-leaf01] error querying nsqlookupd (http://metal.test.metal-stack.io:4161/lookup?topic=switch) - Get http://metal.test.metal-stack.io:4161/lookup?topic=switch: dial tcp: i/o timeout        {\"app\": \"metal-core\"}",
			Level:  zap.DebugLevel,
		},
	}

	for _, tst := range tests {
		tst := tst
		t.Run(tst.Level.String(), func(t *testing.T) {
			// record all messages of all levels
			core, recorded := observer.New(zapcore.DebugLevel)
			logger := zap.New(core)

			bridgeNsqLogToCoreLog(tst.NsqMsg, logger)

			if len(recorded.AllUntimed()) != 1 || recorded.AllUntimed()[0].Level != tst.Level {
				t.Errorf("expected one info level msg")
			}
		})
	}
}
