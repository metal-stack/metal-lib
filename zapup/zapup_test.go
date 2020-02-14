package zapup

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func unsetEnv(keys ...string) {
	for _, k := range keys {
		os.Unsetenv(k)
	}
}

func TestMustRootLoggerSucceeds(t *testing.T) {
	Reset()
	require.NotPanics(t, func() { MustRootLogger() }, "Unexpected panic.")
}

func TestMustRootLoggerPanics(t *testing.T) {
	Reset()

	os.Setenv(KeyLogEncoding, "gopher")
	defer unsetEnv(KeyLogEncoding)
	assert.Panics(t, func() { MustRootLogger() }, "Unexpected success.")
}

func TestEncodingApplies(t *testing.T) {

	temp, err := ioutil.TempFile("", "zapup-encoding-test")
	require.NoError(t, err, "Unexpected error constructing logger.")
	defer os.Remove(temp.Name())

	os.Setenv(KeyLogLevel, "info")
	os.Setenv(KeyFieldCustomer, "gopher")
	os.Setenv(KeyFieldStage, "test")
	os.Setenv(KeyFieldApp, "testapp")
	defer unsetEnv(KeyLogLevel, KeyFieldCustomer, KeyFieldStage, KeyFieldApp)

	tests := []struct {
		enc string
		// verifying the entire log line requires adoptions all the time since e.g. caller line changes on test adoptions
		partial string
	}{
		{"console", `Hello, world!	{"app": "testapp", "customer": "gopher", "stage": "test"}`},
		{"json", `"message":"Hello, world!","app":"testapp","customer":"gopher","stage":"test"`},
	}

	for _, test := range tests {
		Reset()

		os.Setenv(KeyLogEncoding, test.enc)
		os.Setenv(KeyOutput, temp.Name())

		z, err := RootLogger()
		require.NotNil(t, z)
		require.NoError(t, err)

		z.Info("Hello, world!")
		bytes, e := ioutil.ReadAll(temp)
		require.NoError(t, e, "Couldn't read log contents from temp file.")

		assert.Contains(t, string(bytes), test.partial, "Unexpected log.")
		unsetEnv(KeyLogEncoding, KeyOutput)
	}
}

func TestLogsAnnotatedWithFilenameAndLineNumber(t *testing.T) {
	Reset()

	temp, err := ioutil.TempFile("", "zapup-annotation-test")
	require.NoError(t, err, "Unexpected error constructing logger.")
	defer os.Remove(temp.Name())

	os.Setenv(KeyLogLevel, "info")
	os.Setenv(KeyOutput, temp.Name())
	defer unsetEnv(KeyLogLevel, KeyOutput)

	z, err := RootLogger()
	require.NotNil(t, z)
	require.NoError(t, err)

	z.Info("Hello, world!")
	bytes, e := ioutil.ReadAll(temp)
	require.NoError(t, e, "Couldn't read log contents from temp file.")
	logs := string(bytes)

	assert.Contains(t, logs, "\"caller\":\"zapup/zapup_test.go:", "Unexpected or missing caller.")
}

func TestFieldsApplied(t *testing.T) {
	type unit struct {
		app   string
		cust  string
		stage string
	}
	tests := []unit{
		{"", "", ""},
		{"zapup", "gopher", "test"},
	}

	for _, test := range tests {
		temp, err := ioutil.TempFile("", "zapup-fields-applied-test")
		require.NoError(t, err, "Unexpected error constructing logger.")
		defer os.RemoveAll(temp.Name())
		os.Setenv(KeyOutput, temp.Name())

		// verify defaults except the output to be able to test this
		// Level cannot be tested just by expecting the leven inside the log message but by checking the filter.
		os.Setenv(KeyLogLevel, "info")
		os.Setenv(KeyFieldApp, test.app)
		os.Setenv(KeyFieldCustomer, test.cust)
		os.Setenv(KeyFieldStage, test.stage)

		Reset()
		z, err := RootLogger()
		require.NotNil(t, z)
		require.NoError(t, err)

		z.Info("The Go gopher was designed by Renee French.")
		bytes, e := ioutil.ReadAll(temp)
		require.NoError(t, e, "Couldn't read log contents from temp file.")
		logs := string(bytes)

		if test.app == "" && test.cust == "" && test.stage == "" {
			assert.NotContains(t, logs, "\"app\":\"", "Unexpected or missing app.")
			assert.NotContains(t, logs, "\"customer\":\"", "Unexpected or missing customer.")
			assert.NotContains(t, logs, "\"stage\":\"", "Unexpected or missing stage.")
		} else {
			assert.Contains(t, logs, "\"app\":\""+test.app+"\"", "Unexpected or missing app.")
			assert.Contains(t, logs, "\"customer\":\""+test.cust+"\"", "Unexpected or missing customer.")
			assert.Contains(t, logs, "\"stage\":\""+test.stage+"\"", "Unexpected or missing stage.")
		}

		os.Remove(temp.Name())
		unsetEnv(KeyOutput, KeyLogLevel, KeyFieldApp, KeyFieldCustomer, KeyFieldStage)
	}

}

func TestLevelApplies(t *testing.T) {
	type unit struct {
		key string
		val string
		exp zapcore.Level
	}

	tests := []unit{
		{KeyLogLevel, "", zapcore.WarnLevel},
		{KeyLogLevel, "info", zapcore.InfoLevel},
		{KeyLogLevel, "debug", zapcore.DebugLevel},
		{KeyLogLevel, "warn", zapcore.WarnLevel},
		{KeyLogLevel, "error", zapcore.ErrorLevel},
		{KeyLogLevel, "dpanic", zapcore.DPanicLevel},
		{KeyLogLevel, "panic", zapcore.PanicLevel},
		{KeyLogLevel, "fatal", zapcore.FatalLevel},
	}

	for _, test := range tests {
		Reset()

		os.Setenv(test.key, test.val)
		z, err := RootLogger()
		unsetEnv(test.key)

		require.NotNil(t, z)
		require.NoError(t, err)

		c := z.Check(test.exp, "")
		require.NotNil(t, c, "Expectation level applies failed! Wanted: %s.", test.exp)
		require.Equal(t, c.Level, test.exp, "Wanted: %s, Got: %s", c.Level, test.exp)
	}
}
