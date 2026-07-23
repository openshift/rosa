package logging

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/openshift/rosa/pkg/debug"
)

func TestLogging(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Logging Suite")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type stubReadCloser struct {
	data     []byte
	readErr  error
	closeErr error
}

func (s *stubReadCloser) Read(p []byte) (int, error) {
	if s.readErr != nil {
		return 0, s.readErr
	}
	if len(s.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, s.data)
	s.data = s.data[n:]
	if len(s.data) == 0 {
		return n, io.EOF
	}
	return n, nil
}

func (s *stubReadCloser) Close() error {
	return s.closeErr
}

func newDebugLogger(out *bytes.Buffer) *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(out)
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{DisableColors: true, DisableQuote: true})
	return logger
}

func newLoggerWithLevel(out *bytes.Buffer, level logrus.Level) *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(out)
	logger.SetLevel(level)
	logger.SetFormatter(&logrus.TextFormatter{DisableColors: true, DisableQuote: true})
	return logger
}

var _ = Describe("Logging", func() {
	var previousDebug bool

	BeforeEach(func() {
		previousDebug = debug.Enabled()
		debug.SetEnabled(false)
	})

	AfterEach(func() {
		debug.SetEnabled(previousDebug)
	})

	Describe("NewLogger", func() {
		It("creates an info-level logger by default", func() {
			logger := NewLogger()

			Expect(logger.Level).To(Equal(logrus.InfoLevel))
			formatter, ok := logger.Formatter.(*logrus.TextFormatter)
			Expect(ok).To(BeTrue())
			Expect(formatter.DisableColors).To(BeTrue())
			Expect(formatter.DisableQuote).To(BeTrue())
			Expect(formatter.FullTimestamp).To(BeTrue())
		})

		It("creates a debug-level logger when debug mode is enabled", func() {
			debug.SetEnabled(true)

			logger := NewLogger()

			Expect(logger.Level).To(Equal(logrus.DebugLevel))
		})
	})

	Describe("AWSLoggerBuilder", func() {
		It("requires a logger", func() {
			_, err := (&AWSLoggerBuilder{}).Build()
			Expect(err).To(MatchError("Logger is mandatory"))
		})

		It("builds a logger and writes through it", func() {
			buffer := &bytes.Buffer{}
			logger := logrus.New()
			logger.SetOutput(buffer)

			result, err := (&AWSLoggerBuilder{}).Logger(logger).Build()
			Expect(err).NotTo(HaveOccurred())

			result.Log("hello", " ", "aws")
			Expect(buffer.String()).To(ContainSubstring("hello aws"))
		})
	})

	Describe("OCMLoggerBuilder", func() {
		It("requires a logger", func() {
			_, err := NewOCMLogger().Build()
			Expect(err).To(MatchError("Logger is mandatory"))
		})

		It("builds a logger with level helpers", func() {
			logger := logrus.New()
			logger.SetLevel(logrus.InfoLevel)

			result, err := NewOCMLogger().Logger(logger).Build()
			Expect(err).NotTo(HaveOccurred())

			Expect(result.DebugEnabled()).To(BeFalse())
			Expect(result.InfoEnabled()).To(BeTrue())
			Expect(result.WarnEnabled()).To(BeTrue())
			Expect(result.ErrorEnabled()).To(BeTrue())
		})

		It("routes info messages through debug logging", func() {
			logBuffer := &bytes.Buffer{}
			result, err := NewOCMLogger().Logger(newLoggerWithLevel(logBuffer, logrus.InfoLevel)).Build()
			Expect(err).NotTo(HaveOccurred())

			result.Info(context.Background(), "info message %s", "value")

			Expect(logBuffer.String()).To(BeEmpty())
		})

		It("routes warn messages through debug logging", func() {
			logBuffer := &bytes.Buffer{}
			result, err := NewOCMLogger().Logger(newLoggerWithLevel(logBuffer, logrus.WarnLevel)).Build()
			Expect(err).NotTo(HaveOccurred())

			result.Warn(context.Background(), "warn message %s", "value")

			Expect(logBuffer.String()).To(BeEmpty())
		})

		It("routes debug messages through debug logging", func() {
			logBuffer := &bytes.Buffer{}
			result, err := NewOCMLogger().Logger(newLoggerWithLevel(logBuffer, logrus.DebugLevel)).Build()
			Expect(err).NotTo(HaveOccurred())

			result.Debug(context.Background(), "debug message %s", "value")

			Expect(logBuffer.String()).To(ContainSubstring("debug message value"))
		})

		It("routes error messages through error logging", func() {
			logBuffer := &bytes.Buffer{}
			result, err := NewOCMLogger().Logger(newLoggerWithLevel(logBuffer, logrus.ErrorLevel)).Build()
			Expect(err).NotTo(HaveOccurred())

			result.Error(context.Background(), "error message %s", "value")

			Expect(logBuffer.String()).To(ContainSubstring("error message value"))
		})

		It("routes fatal messages through error logging without exiting", func() {
			logBuffer := &bytes.Buffer{}
			result, err := NewOCMLogger().Logger(newLoggerWithLevel(logBuffer, logrus.ErrorLevel)).Build()
			Expect(err).NotTo(HaveOccurred())

			result.Fatal(context.Background(), "fatal message %s", "value")

			Expect(logBuffer.String()).To(ContainSubstring("fatal message value"))
		})
	})

	Describe("RoundTripperBuilder", func() {
		It("requires a logger", func() {
			_, err := NewRoundTripper().
				Next(roundTripFunc(func(request *http.Request) (*http.Response, error) {
					return nil, nil
				})).
				Build()
			Expect(err).To(MatchError("Logger is mandatory"))
		})

		It("requires a next handler", func() {
			_, err := NewRoundTripper().
				Logger(logrus.New()).
				Build()
			Expect(err).To(MatchError("Next handler is mandatory"))
		})

		It("copies the redact configuration from the builder", func() {
			builder := NewRoundTripper().
				Logger(logrus.New()).
				Redact("token").
				Next(roundTripFunc(func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Body:       io.NopCloser(strings.NewReader("ok")),
						Header:     http.Header{},
					}, nil
				}))

			result, err := builder.Build()
			Expect(err).NotTo(HaveOccurred())

			builder.Redact("other")
			Expect(result.redact).To(HaveKeyWithValue("token", true))
			Expect(result.redact).NotTo(HaveKey("other"))
		})
	})

	Describe("RoundTripper", func() {
		It("returns request body read errors without calling the next handler", func() {
			nextCalled := false
			roundTripper, err := NewRoundTripper().
				Logger(newDebugLogger(&bytes.Buffer{})).
				Next(roundTripFunc(func(request *http.Request) (*http.Response, error) {
					nextCalled = true
					return nil, nil
				})).
				Build()
			Expect(err).NotTo(HaveOccurred())

			request, err := http.NewRequest(http.MethodPost, "https://example.com", nil)
			Expect(err).NotTo(HaveOccurred())
			request.Body = &stubReadCloser{readErr: errors.New("request read failed")}

			_, err = roundTripper.RoundTrip(request)
			Expect(err).To(MatchError("request read failed"))
			Expect(nextCalled).To(BeFalse())
		})

		It("returns request body close errors without calling the next handler", func() {
			nextCalled := false
			roundTripper, err := NewRoundTripper().
				Logger(newDebugLogger(&bytes.Buffer{})).
				Next(roundTripFunc(func(request *http.Request) (*http.Response, error) {
					nextCalled = true
					return nil, nil
				})).
				Build()
			Expect(err).NotTo(HaveOccurred())

			request, err := http.NewRequest(http.MethodPost, "https://example.com", nil)
			Expect(err).NotTo(HaveOccurred())
			request.Body = &stubReadCloser{
				data:     []byte("request-body"),
				closeErr: errors.New("request close failed"),
			}

			_, err = roundTripper.RoundTrip(request)
			Expect(err).To(MatchError("request close failed"))
			Expect(nextCalled).To(BeFalse())
		})

		It("returns errors from the next handler when the request has no body", func() {
			roundTripper, err := NewRoundTripper().
				Logger(newDebugLogger(&bytes.Buffer{})).
				Next(roundTripFunc(func(request *http.Request) (*http.Response, error) {
					Expect(request.Body).To(BeNil())
					return nil, errors.New("next failed")
				})).
				Build()
			Expect(err).NotTo(HaveOccurred())

			request, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
			Expect(err).NotTo(HaveOccurred())

			_, err = roundTripper.RoundTrip(request)
			Expect(err).To(MatchError("next failed"))
		})

		It("returns response body read errors", func() {
			roundTripper, err := NewRoundTripper().
				Logger(newDebugLogger(&bytes.Buffer{})).
				Next(roundTripFunc(func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Header:     http.Header{"Content-Type": []string{"application/json"}},
						Body:       &stubReadCloser{readErr: errors.New("response read failed")},
					}, nil
				})).
				Build()
			Expect(err).NotTo(HaveOccurred())

			request, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
			Expect(err).NotTo(HaveOccurred())

			_, err = roundTripper.RoundTrip(request)
			Expect(err).To(MatchError("response read failed"))
		})

		It("returns response body close errors", func() {
			roundTripper, err := NewRoundTripper().
				Logger(newDebugLogger(&bytes.Buffer{})).
				Next(roundTripFunc(func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Header:     http.Header{"Content-Type": []string{"application/json"}},
						Body: &stubReadCloser{
							data:     []byte(`{"visible":"response-value"}`),
							closeErr: errors.New("response close failed"),
						},
					}, nil
				})).
				Build()
			Expect(err).NotTo(HaveOccurred())

			request, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
			Expect(err).NotTo(HaveOccurred())

			_, err = roundTripper.RoundTrip(request)
			Expect(err).To(MatchError("response close failed"))
		})

		It("handles responses without a body", func() {
			roundTripper, err := NewRoundTripper().
				Logger(newDebugLogger(&bytes.Buffer{})).
				Next(roundTripFunc(func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusNoContent,
						Status:     "204 No Content",
						Header:     http.Header{},
					}, nil
				})).
				Build()
			Expect(err).NotTo(HaveOccurred())

			request, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
			Expect(err).NotTo(HaveOccurred())

			response, err := roundTripper.RoundTrip(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Body).To(BeNil())
		})

		It("preserves request and response bodies through the round trip", func() {
			var capturedRequestBody string
			roundTripper, err := NewRoundTripper().
				Logger(newDebugLogger(&bytes.Buffer{})).
				Redact("token").
				Next(roundTripFunc(func(request *http.Request) (*http.Response, error) {
					body, readErr := io.ReadAll(request.Body)
					Expect(readErr).NotTo(HaveOccurred())
					capturedRequestBody = string(body)
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Header:     http.Header{"Content-Type": []string{"application/json"}},
						Body:       io.NopCloser(strings.NewReader(`{"visible":"response-value"}`)),
					}, nil
				})).
				Build()
			Expect(err).NotTo(HaveOccurred())

			request, err := http.NewRequest(http.MethodPost, "https://example.com", strings.NewReader(`{"visible":"request-value"}`))
			Expect(err).NotTo(HaveOccurred())
			request.Header.Set("Content-Type", "application/json")

			response, err := roundTripper.RoundTrip(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(capturedRequestBody).To(Equal(`{"visible":"request-value"}`))

			responseBody, err := io.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(responseBody)).To(Equal(`{"visible":"response-value"}`))
		})

		It("redacts configured JSON fields in logs", func() {
			logBuffer := &bytes.Buffer{}
			roundTripper, err := NewRoundTripper().
				Logger(newDebugLogger(logBuffer)).
				Redact("token").
				Next(roundTripFunc(func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Header:     http.Header{"Content-Type": []string{"application/json"}},
						Body:       io.NopCloser(strings.NewReader(`{"token":"response-secret","visible":"response-visible"}`)),
					}, nil
				})).
				Build()
			Expect(err).NotTo(HaveOccurred())

			request, err := http.NewRequest(http.MethodPost, "https://example.com", strings.NewReader(`{"token":"request-secret","visible":"request-visible"}`))
			Expect(err).NotTo(HaveOccurred())
			request.Header.Set("Content-Type", "application/json")

			_, err = roundTripper.RoundTrip(request)
			Expect(err).NotTo(HaveOccurred())

			logOutput := logBuffer.String()
			Expect(logOutput).To(ContainSubstring("***"))
			Expect(logOutput).To(ContainSubstring("request-visible"))
			Expect(logOutput).To(ContainSubstring("response-visible"))
			Expect(logOutput).NotTo(ContainSubstring("request-secret"))
			Expect(logOutput).NotTo(ContainSubstring("response-secret"))
		})

		It("pretty prints JSON bodies in logs", func() {
			logBuffer := &bytes.Buffer{}
			roundTripper, err := NewRoundTripper().
				Logger(newDebugLogger(logBuffer)).
				Next(roundTripFunc(func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Header:     http.Header{"Content-Type": []string{"application/json"}},
						Body: io.NopCloser(strings.NewReader(
							`{"visible":"response-value","nested":{"name":"response-child"}}`,
						)),
					}, nil
				})).
				Build()
			Expect(err).NotTo(HaveOccurred())

			request, err := http.NewRequest(
				http.MethodPost,
				"https://example.com",
				strings.NewReader(`{"visible":"request-value","nested":{"name":"request-child"}}`),
			)
			Expect(err).NotTo(HaveOccurred())
			request.Header.Set("Content-Type", "application/json")

			_, err = roundTripper.RoundTrip(request)
			Expect(err).NotTo(HaveOccurred())

			logOutput := logBuffer.String()
			Expect(logOutput).To(ContainSubstring("{\n  \"nested\": {\n    \"name\": \"request-child\"\n  },\n  \"visible\": \"request-value\"\n}"))
			Expect(logOutput).To(ContainSubstring("{\n  \"nested\": {\n    \"name\": \"response-child\"\n  },\n  \"visible\": \"response-value\"\n}"))
		})

		It("falls back to raw bytes for invalid JSON bodies", func() {
			logBuffer := &bytes.Buffer{}
			roundTripper, err := NewRoundTripper().
				Logger(newDebugLogger(logBuffer)).
				Next(roundTripFunc(func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Header:     http.Header{},
						Body:       io.NopCloser(strings.NewReader("ok")),
					}, nil
				})).
				Build()
			Expect(err).NotTo(HaveOccurred())

			request, err := http.NewRequest(http.MethodPost, "https://example.com", strings.NewReader("{invalid-json"))
			Expect(err).NotTo(HaveOccurred())
			request.Header.Set("Content-Type", "application/json")

			_, err = roundTripper.RoundTrip(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(logBuffer.String()).To(ContainSubstring("{invalid-json"))
		})

		It("falls back to raw bytes when JSON body contains multiple top-level values", func() {
			logBuffer := &bytes.Buffer{}
			roundTripper, err := NewRoundTripper().
				Logger(newDebugLogger(logBuffer)).
				Next(roundTripFunc(func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Header:     http.Header{},
						Body:       io.NopCloser(strings.NewReader("ok")),
					}, nil
				})).
				Build()
			Expect(err).NotTo(HaveOccurred())

			request, err := http.NewRequest(
				http.MethodPost,
				"https://example.com",
				strings.NewReader(`{"a":"1"}{"b":"2"}`),
			)
			Expect(err).NotTo(HaveOccurred())
			request.Header.Set("Content-Type", "application/json")

			_, err = roundTripper.RoundTrip(request)
			Expect(err).NotTo(HaveOccurred())

			logOutput := logBuffer.String()
			Expect(logOutput).To(ContainSubstring(`{"a":"1"}{"b":"2"}`))
		})

		It("omits the Authorization header from logs", func() {
			logBuffer := &bytes.Buffer{}
			roundTripper, err := NewRoundTripper().
				Logger(newDebugLogger(logBuffer)).
				Next(roundTripFunc(func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Header:     http.Header{},
						Body:       io.NopCloser(strings.NewReader("ok")),
					}, nil
				})).
				Build()
			Expect(err).NotTo(HaveOccurred())

			request, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
			Expect(err).NotTo(HaveOccurred())
			request.Header.Set("Authorization", "Bearer super-secret")

			_, err = roundTripper.RoundTrip(request)
			Expect(err).NotTo(HaveOccurred())

			logOutput := logBuffer.String()
			Expect(logOutput).To(ContainSubstring("Request header 'Authorization' is omitted"))
			Expect(logOutput).NotTo(ContainSubstring("super-secret"))
		})

		It("redacts configured form fields in logs", func() {
			logBuffer := &bytes.Buffer{}
			roundTripper, err := NewRoundTripper().
				Logger(newDebugLogger(logBuffer)).
				Redact("token").
				Next(roundTripFunc(func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Header:     http.Header{},
						Body:       io.NopCloser(strings.NewReader("ok")),
					}, nil
				})).
				Build()
			Expect(err).NotTo(HaveOccurred())

			request, err := http.NewRequest(http.MethodPost, "https://example.com", strings.NewReader("token=secret&name=value"))
			Expect(err).NotTo(HaveOccurred())
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			_, err = roundTripper.RoundTrip(request)
			Expect(err).NotTo(HaveOccurred())

			logOutput := logBuffer.String()
			Expect(logOutput).To(ContainSubstring("***"))
			Expect(logOutput).To(ContainSubstring("name=value"))
			Expect(logOutput).NotTo(ContainSubstring("secret"))
		})

		It("omits malformed form bodies instead of logging raw secret values", func() {
			logBuffer := &bytes.Buffer{}
			roundTripper, err := NewRoundTripper().
				Logger(newDebugLogger(logBuffer)).
				Redact("token").
				Next(roundTripFunc(func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Header:     http.Header{},
						Body:       io.NopCloser(strings.NewReader("ok")),
					}, nil
				})).
				Build()
			Expect(err).NotTo(HaveOccurred())

			request, err := http.NewRequest(http.MethodPost, "https://example.com", strings.NewReader("token=%zz"))
			Expect(err).NotTo(HaveOccurred())
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			_, err = roundTripper.RoundTrip(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(logBuffer.String()).To(ContainSubstring("Request body omitted due to invalid form encoding"))
			Expect(logBuffer.String()).NotTo(ContainSubstring("token=%zz"))
		})

		It("falls back to raw bytes when the content type is malformed", func() {
			logBuffer := &bytes.Buffer{}
			roundTripper, err := NewRoundTripper().
				Logger(newDebugLogger(logBuffer)).
				Next(roundTripFunc(func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Header:     http.Header{},
						Body:       io.NopCloser(strings.NewReader("ok")),
					}, nil
				})).
				Build()
			Expect(err).NotTo(HaveOccurred())

			request, err := http.NewRequest(http.MethodPost, "https://example.com", strings.NewReader("raw-body"))
			Expect(err).NotTo(HaveOccurred())
			request.Header.Set("Content-Type", "invalid;=")

			_, err = roundTripper.RoundTrip(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(logBuffer.String()).To(ContainSubstring("Failed to parse content type"))
			Expect(logBuffer.String()).To(ContainSubstring("raw-body"))
		})
	})
})
