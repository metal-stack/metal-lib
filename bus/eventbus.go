package bus

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"time"

	"go.uber.org/zap"

	"github.com/nsqio/go-nsq"
)

// A TLSConfig represents the TLS config of an NSQ publisher/consumer.
type TLSConfig struct {
	CACertFile     string
	ClientCertFile string
}

// Inactive a TLSConfig considered inactive if neither a ca-cert nor a client-cert is present
func (cfg *TLSConfig) Inactive() bool {
	return cfg == nil || len(cfg.CACertFile) == 0 || len(cfg.ClientCertFile) == 0
}

// A PublisherConfig represents the config of an NSQ publisher.
type PublisherConfig struct {
	TCPAddress   string
	HTTPEndpoint string
	TLS          *TLSConfig
	NSQ          *nsq.Config
}

// A Receiver is a callback when you receive messages from the bus.
type Receiver func(interface{}) error

// A Consumer wraps the base configuration for the nsq connection
type Consumer struct {
	lookupds []string
	nsqds    []string
	config   *nsq.Config
	log      *zap.Logger
	logLevel nsq.LogLevel
}

type ConsumerRegistration struct {
	consumer  *Consumer
	log       *zap.Logger
	c         *nsq.Consumer
	connected bool

	timeout   time.Duration
	onTimeout OnTimeout

	// time to live for message in nanos
	ttl time.Duration
}

type Option func(registration *Consumer) *Consumer

// Level specifies the severity of a given log message
type Level int

// Log levels
const (
	Debug Level = iota
	Info
	Warning
	Error
)

// LogLevel maps between our loglevel and nsq loglevels
func LogLevel(v Level) Option {
	return func(c *Consumer) *Consumer {
		var l nsq.LogLevel
		switch v {
		case Debug:
			l = nsq.LogLevelDebug
		case Info:
			l = nsq.LogLevelInfo
		case Warning:
			l = nsq.LogLevelWarning
		case Error:
			l = nsq.LogLevelError
		default:
			l = nsq.LogLevelWarning
		}

		c.logLevel = l

		return c
	}
}

func NSQDs(nsqds ...string) Option {
	return func(c *Consumer) *Consumer {
		c.nsqds = nsqds
		return c
	}
}

func MaxInFlight(num int) Option {
	return func(c *Consumer) *Consumer {
		c.config.MaxInFlight = num
		return c
	}
}

// NewConsumer returns a consumer and stores the addresses of the lookupd's.
func NewConsumer(log *zap.Logger, tlsCfg *TLSConfig, lookupds ...string) (*Consumer, error) {
	cfg := CreateNSQConfig(tlsCfg)
	cfg.LookupdPollInterval = time.Second * 5
	cfg.HeartbeatInterval = time.Second * 5
	cfg.DefaultRequeueDelay = time.Second * 5
	cfg.MaxInFlight = 10

	return &Consumer{
		config:   cfg,
		lookupds: lookupds,
		log:      log,
		logLevel: nsq.LogLevelInfo,
	}, nil
}

func (c *Consumer) With(opts ...Option) *Consumer {
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Consumer) MustRegister(topic, channel string) *ConsumerRegistration {
	cr, err := c.Register(topic, channel)
	if err != nil {
		panic(err)
	}
	return cr
}

func (c *Consumer) Register(topic, channel string) (*ConsumerRegistration, error) {
	q, err := nsq.NewConsumer(topic, channel, c.config)
	if err != nil {
		return nil, fmt.Errorf("cannot create consumer for topic:%q, channel:%q: %v", topic, channel, err)
	}

	cr := &ConsumerRegistration{
		consumer: c,
		log:      c.log,
		c:        q,
	}

	return cr, nil
}

func (cr *ConsumerRegistration) Output(num int, msg string) error {
	bridgeNsqLogToCoreLog(msg, cr.log)
	return nil
}

type TimeoutError struct {
	event interface{}
}

func (t TimeoutError) Error() string {
	return "Timeout processing event"
}

func (t TimeoutError) Event() interface{} {
	return t.event
}

// OnTimeout function that is called in the case of timeout while handling the event
type OnTimeout func(err TimeoutError) error

type timeoutWrapper struct {
	// time to live for message in nanos
	ttl       time.Duration
	timeout   time.Duration
	onTimeout OnTimeout
	msgType   reflect.Type
	recv      Receiver
	log       *zap.Logger
}

func (tw *timeoutWrapper) handleWithTimeout(message *nsq.Message) error {

	if tw.ttl > 0 {
		// calculate the age of the message in nanos
		t1 := message.Timestamp
		t2 := time.Now().UnixNano()
		ageNanos := time.Duration(t2 - t1)

		if ageNanos > tw.ttl {
			if tw.log != nil {
				tw.log.Warn("Dropped message", zap.String("id", string(message.ID[:])), zap.Duration("ageNanos", ageNanos))
			}

			// drop message
			return nil
		}
	}

	newval := reflect.New(tw.msgType)
	nv := newval.Elem().Addr().Interface()
	err := json.Unmarshal(message.Body, nv)
	if err != nil {
		return err
	}

	// timeout == 0 means synchronous call without timeout
	if tw.timeout == 0 {
		return tw.recv(nv)
	}

	c1 := make(chan error, 1)
	go func() {
		c1 <- tw.recv(nv)
	}()

	select {
	case err = <-c1:
		return err
	case <-time.After(tw.timeout):

		if tw.onTimeout != nil {
			return tw.onTimeout(TimeoutError{
				event: nv,
			})
		}

		return nil
	}
}

type crOption func(registration *ConsumerRegistration) *ConsumerRegistration

// Timeout guards the event handler with a timeout, timeout 0 means no timeout.
// The optional timeoutFunction is called in the case of timeout while handling the event.
func Timeout(timeout time.Duration, timeoutFunction OnTimeout) crOption {
	return func(cr *ConsumerRegistration) *ConsumerRegistration {
		cr.timeout = timeout
		cr.onTimeout = timeoutFunction
		return cr
	}
}

// TTL specifies the maximum age of messages to accept.
// If a message is received that is older than the given ttl, it will be dropped.
func TTL(ttl time.Duration) crOption {
	return func(cr *ConsumerRegistration) *ConsumerRegistration {
		cr.ttl = ttl
		return cr
	}
}

// Consume a message
func (cr *ConsumerRegistration) Consume(paramProto interface{}, recv Receiver, concurrent int, opts ...crOption) error {
	if cr.connected {
		return fmt.Errorf("already connected")
	}

	for _, opt := range opts {
		opt(cr)
	}

	tp := reflect.TypeOf(paramProto)
	tw := &timeoutWrapper{
		msgType:   tp,
		recv:      recv,
		onTimeout: cr.onTimeout,
		timeout:   cr.timeout,
		ttl:       cr.ttl,
		log:       cr.log,
	}

	cr.c.SetLogger(cr, cr.consumer.logLevel)
	cr.c.AddConcurrentHandlers(nsq.HandlerFunc(tw.handleWithTimeout), concurrent)
	cr.connected = true

	if cr.consumer.nsqds != nil {
		return cr.c.ConnectToNSQDs(cr.consumer.nsqds)
	}
	return cr.c.ConnectToNSQLookupds(cr.consumer.lookupds)
}

// A Publisher is used for event publishing to topics. The fields
// Publish and CreateTopics can be overwritten to mock this publisher.
type Publisher interface {
	Publish(topic string, data interface{}) error
	CreateTopic(topic string) error
}

type nsqPublisher struct {
	log          *zap.Logger
	producer     *nsq.Producer
	httpEndpoint string
	client       *http.Client
}

func (p *nsqPublisher) Output(num int, msg string) error {
	bridgeNsqLogToCoreLog(msg, p.log)
	return nil
}

// NewPublisher creates a new publisher to produce events for topics.
func NewPublisher(zlog *zap.Logger, publisherCfg *PublisherConfig) (Publisher, error) {
	publisherCfg.ConfigureNSQ()
	p, err := nsq.NewProducer(publisherCfg.TCPAddress, publisherCfg.NSQ)
	if err != nil {
		return nil, fmt.Errorf("cannot create producer with nsqd=%q: %v", publisherCfg.TCPAddress, err)
	}
	pbl := &nsqPublisher{
		log:          zlog,
		producer:     p,
		httpEndpoint: publisherCfg.HTTPEndpoint,
		client:       http.DefaultClient,
	}

	p.SetLogger(pbl, nsq.LogLevelError)
	return pbl, nil
}

// Publish posts the given data as a json string into the topic.
func (p *nsqPublisher) Publish(topic string, data interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("cannot marshal data to json: %v", err)
	}
	return p.producer.Publish(topic, b)
}

// CreateTopic needs to be called with an HTTP request since the library does not support creating
// topics (they are created implicitly)
func (p *nsqPublisher) CreateTopic(topic string) error {
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/topic/create?topic=%s", p.httpEndpoint, topic), nil)
	if err != nil {
		return err
	}
	req.Header.Add("ContentType", "text/plain")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 300 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("error creating topic: %s", string(bodyBytes))
	}

	_ = resp.Body.Close()
	return nil
}

// understands the nsq log message and writes it in the zap.logger
//
// Format:
// INF    2 [switch/fra-equ01-leaf01] querying nsqlookupd http://metal.test.metal-stack.io:4161/lookup?topic=switch      {"app": "metal-core"}
func bridgeNsqLogToCoreLog(nsqLogMessage string, log *zap.Logger) {
	logLevel := nsqLogMessage[:3]
	logMessage := nsqLogMessage[5:]

	switch logLevel {
	case nsq.LogLevelError.String():
		log.Error(logMessage)
	case nsq.LogLevelWarning.String():
		log.Warn(logMessage)
	case nsq.LogLevelInfo.String():
		log.Info(logMessage)
	case nsq.LogLevelDebug.String():
		log.Debug(logMessage)
	default:
		log.Info(logMessage)
	}
}
