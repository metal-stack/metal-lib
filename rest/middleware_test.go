package rest

// func TestRequestLoggerFilter(t *testing.T) {
// 	type logMessage struct {
// 		Level         string `json:"level"`
// 		RequestID     string `json:"rqid"`
// 		Message       string `json:"msg"`
// 		RemoteAddr    string `json:"remoteaddr"`
// 		Method        string `json:"method"`
// 		URI           string `json:"uri"`
// 		Route         string `json:"route"`
// 		Status        int    `json:"status"`
// 		ContentLength int    `json:"content-length"`
// 		Duration      string `json:"duration"`
// 		Body          string `json:"body"`
// 		Response      string `json:"response"`
// 	}

// 	tests := []struct {
// 		name           string
// 		level          slog.Level
// 		handler        func(req *restful.Request, resp *restful.Response)
// 		wantRequestLog *logMessage
// 		wantClosingLog *logMessage
// 		wantBody       bool
// 	}{
// 		{
// 			name:  "info level",
// 			level: slog.LevelInfo,
// 			handler: func(req *restful.Request, resp *restful.Response) {
// 				requestLogger := GetLoggerFromContext(req.Request, nil)
// 				requestLogger.Info("this is a test message")
// 				_ = resp.WriteHeaderAndEntity(http.StatusOK, nil)
// 			},
// 			wantRequestLog: &logMessage{
// 				Level:      "info",
// 				Message:    "this is a test message",
// 				RemoteAddr: "1.2.3.4",
// 				Method:     "GET",
// 				URI:        "/test",
// 				Route:      "/test",
// 			},
// 			wantClosingLog: &logMessage{
// 				Level:      "info",
// 				Message:    "finished handling rest call",
// 				RemoteAddr: "1.2.3.4",
// 				Method:     "GET",
// 				URI:        "/test",
// 				Route:      "/test",
// 				Status:     http.StatusOK,
// 			},
// 		},
// 		{
// 			name:  "debug level",
// 			level: slog.LevelDebug,
// 			handler: func(req *restful.Request, resp *restful.Response) {
// 				requestLogger := GetLoggerFromContext(req.Request, nil)
// 				requestLogger.Debug("this is a test message")
// 				_ = resp.WriteHeaderAndEntity(http.StatusOK, "Test Response")
// 			},
// 			wantRequestLog: &logMessage{
// 				Level:      "debug",
// 				Message:    "this is a test message",
// 				RemoteAddr: "1.2.3.4",
// 				Method:     "GET",
// 				URI:        "/test",
// 				Route:      "/test",
// 			},
// 			wantClosingLog: &logMessage{
// 				Level:         "info",
// 				Message:       "finished handling rest call",
// 				RemoteAddr:    "1.2.3.4",
// 				Method:        "GET",
// 				URI:           "/test",
// 				Route:         "/test",
// 				Status:        http.StatusOK,
// 				ContentLength: 15,
// 				Response:      `"Test Response"`,
// 			},
// 			wantBody: true,
// 		},
// 	}
// 	for i := range tests {
// 		tt := tests[i]
// 		t.Run(tt.name, func(t *testing.T) {
// 			testLogger := slog.Default()
// 			log := testLogger.GetLogger().WithGroup("test-logger")

// 			sendRequestThroughFilterChain(t, tt.handler, RequestLoggerFilter(log))

// 			lines := strings.Split(testLogger.GetLogs(), "\n")
// 			t.Log(lines)

// 			require.Len(t, lines, 2)

// 			var requestLog logMessage
// 			err := json.Unmarshal([]byte(lines[0]), &requestLog)
// 			require.NoError(t, err)

// 			assert.NotEmpty(t, requestLog.RequestID)
// 			_, err = uuid.Parse(requestLog.RequestID)
// 			require.NoError(t, err)

// 			if tt.wantBody {
// 				assert.NotEmpty(t, requestLog.Body)
// 			}

// 			if diff := cmp.Diff(&requestLog, tt.wantRequestLog, cmpopts.IgnoreFields(logMessage{}, "RequestID", "Body")); diff != "" {
// 				t.Errorf("diff in entry log: %s", diff)
// 			}

// 			var closingLog logMessage
// 			err = json.Unmarshal([]byte(lines[1]), &closingLog)
// 			require.NoError(t, err)

// 			assert.NotEmpty(t, closingLog.RequestID)
// 			_, err = uuid.Parse(closingLog.RequestID)
// 			require.NoError(t, err)

// 			d, err := time.ParseDuration(closingLog.Duration)
// 			require.NoError(t, err)
// 			assert.Greater(t, int64(d), int64(0))

// 			if tt.wantBody {
// 				assert.NotEmpty(t, closingLog.Body)
// 			}

// 			if diff := cmp.Diff(&closingLog, tt.wantClosingLog, cmpopts.IgnoreFields(logMessage{}, "RequestID", "Duration", "Body")); diff != "" {
// 				t.Errorf("diff in closing log: %s", diff)
// 			}
// 		})
// 	}
// }

// type ZapTestLogger struct {
// 	io.Writer

// 	b       *bytes.Buffer
// 	bwriter *bufio.Writer
// 	logger  *slog.Logger
// }

// func (z ZapTestLogger) Close() error {
// 	return nil
// }

// func (z ZapTestLogger) Sync() error {
// 	return nil
// }

// func (z *ZapTestLogger) GetLogs() string {
// 	z.bwriter.Flush()
// 	return strings.TrimSpace(z.b.String())
// }

// func (z *ZapTestLogger) GetLogger() *slog.Logger {
// 	return z.logger
// }

// func sendRequestThroughFilterChain(t *testing.T, handler func(req *restful.Request, resp *restful.Response), filters ...restful.FilterFunction) {
// 	ws := new(restful.WebService).Path("/").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

// 	c := restful.NewContainer()
// 	c.Add(ws.Route(ws.GET("test").To(handler)))
// 	for _, f := range filters {
// 		c.Filter(f)
// 	}

// 	httpRequest, err := http.NewRequestWithContext(context.TODO(), "GET", "http://localhost/test", nil)
// 	require.NoError(t, err)
// 	httpRequest.RemoteAddr = "1.2.3.4"
// 	httpRequest.Header.Set("Accept", "application/json")

// 	httpWriter := httptest.NewRecorder()

// 	c.Dispatch(httpWriter, httpRequest)

// 	require.Equal(t, http.StatusOK, httpWriter.Code)
// }
