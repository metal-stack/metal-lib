package zapup

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// KeyLogLevel sets log level. Valid values: "info", "debug", "warn", "error", "dpanic", "panic", "fatal".
	KeyLogLevel string = "ZAP_LEVEL"
	// KeyLogEncoding sets log output encoding. Valid values: "console", "json".
	KeyLogEncoding string = "ZAP_ENCODING"
	// KeyFieldCustomer sets  meta information for field "customer" to use within log analytics.
	KeyFieldCustomer string = "ZAP_CUSTOMER"
	// KeyFieldApp sets meta information for a field "app" to use within log analytics. Defaults to empty string.
	KeyFieldApp string = "ZAP_APP"
	// KeyFieldStage sets meta information for a field "stage" to use within log analytics. Defaults to empty string.
	KeyFieldStage string = "ZAP_STAGE"
	// KeyOutput sets the output path for zap. Valid values: "stdout", "stderr", "<file path>". Defaults to empty string.
	KeyOutput string = "ZAP_OUT"
)

var (
	once   = new(sync.Once)
	logger *zap.Logger
)

// MustRootLogger returns the inited logger or panics.
func MustRootLogger() *zap.Logger {
	if _, err := RootLogger(); err != nil {
		panic(fmt.Sprintf("Root logger failed: %v.", err))
	}
	return logger
}

// RootLogger initiates and returns the root logger considering environment variables: ZAP_LEVEL, ZAP_ENCODING, ZAP_CUSTOMER, ZAP_APP, ZAP_OUT.
// init() is explicitly NOT(!) used to avoid issues with writing logs by other init() functions and init races.
func RootLogger() (*zap.Logger, error) {
	var err error
	// Run only once to have a singleton root logger.
	once.Do(func() {
		logger, err = newLogger(logLevel(KeyLogLevel, "warn"),
			logEncoding(KeyLogEncoding, "json"),
			logOutput(KeyOutput, "stdout"),
			initialField(KeyFieldApp, "app"),
			initialField(KeyFieldStage, "stage"),
			initialField(KeyFieldCustomer, "customer"))
	})
	return logger, err
}

// Reset resets the Root Logger to enable creating it again on changed environment.
func Reset() {
	once = new(sync.Once)
}

func initialField(env, key string) func(*zap.Config) {
	return func(c *zap.Config) {
		if c.InitialFields == nil {
			c.InitialFields = make(map[string]interface{})
		}
		val := getenvOr(env, "")
		val = purify(val)
		if val == "" {
			// we agreed on having no fields for empty values
			return
		}
		c.InitialFields[key] = purify(val)
	}
}

func logOutput(key, fallback string) func(*zap.Config) {
	return func(c *zap.Config) {
		path := getenvOr(key, fallback)
		path = purify(path)
		c.OutputPaths = []string{path}
		c.ErrorOutputPaths = c.OutputPaths
	}
}

func logEncoding(key, fallback string) func(*zap.Config) {
	return func(c *zap.Config) {
		enc := getenvOr(key, fallback)
		c.Encoding = purify(enc)
	}
}

func logLevel(key, fallback string) func(*zap.Config) {
	return func(c *zap.Config) {
		val := getenvOr(key, fallback)
		val = purify(val)
		level := zap.NewAtomicLevel()
		err := level.UnmarshalText([]byte(val))
		if err != nil {
			panic(fmt.Sprintf("Error unmarshaling level: %v", err))
		}
		c.Level = level
	}
}

func newLogger(options ...func(*zap.Config)) (*zap.Logger, error) {
	config := &zap.Config{}
	for _, option := range options {
		option(config)
	}

	config.Development = false
	config.EncoderConfig = zap.NewProductionEncoderConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.TimeKey = "time"
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.StacktraceKey = ""
	config.Sampling = &zap.SamplingConfig{
		Initial:    100,
		Thereafter: 100,
	}

	return config.Build(zap.AddCaller())
}

func purify(value string) string {
	res := strings.ToLower(value)
	res = strings.TrimSpace(res)
	return res
}

func getenvOr(key, defaultValue string) string {
	res := os.Getenv(key)
	if res == "" {
		res = defaultValue
	}
	return res
}

type key int

var (
	logkey = key(0)
)

// RequestLogger returns the request logger from the request.
func RequestLogger(rq *http.Request) *zap.Logger {
	l, ok := rq.Context().Value(logkey).(*zap.Logger)
	if ok {
		return l
	}
	return MustRootLogger()
}

func PutLogger(ctx context.Context, lg *zap.Logger) context.Context {
	return context.WithValue(ctx, logkey, lg)
}
