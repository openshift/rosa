package logging

import (
	"bytes"
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
		It("preserves request and response bodies and redacts JSON fields in logs", func() {
			logBuffer := &bytes.Buffer{}
			logger := logrus.New()
			logger.SetOutput(logBuffer)
			logger.SetLevel(logrus.DebugLevel)
			logger.SetFormatter(&logrus.TextFormatter{DisableColors: true, DisableQuote: true})

			var capturedRequestBody string
			roundTripper, err := NewRoundTripper().
				Logger(logger).
				Redact("token").
				Next(roundTripFunc(func(request *http.Request) (*http.Response, error) {
					body, readErr := io.ReadAll(request.Body)
					Expect(readErr).NotTo(HaveOccurred())
					capturedRequestBody = string(body)
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
			request.Header.Set("Authorization", "Bearer super-secret")
			request.Header.Set("Content-Type", "application/json")

			response, err := roundTripper.RoundTrip(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(capturedRequestBody).To(Equal(`{"token":"request-secret","visible":"request-visible"}`))

			responseBody, err := io.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(responseBody)).To(Equal(`{"token":"response-secret","visible":"response-visible"}`))

			logOutput := logBuffer.String()
			Expect(logOutput).To(ContainSubstring("Request header 'Authorization' is omitted"))
			Expect(logOutput).To(ContainSubstring("***"))
			Expect(logOutput).To(ContainSubstring("request-visible"))
			Expect(logOutput).To(ContainSubstring("response-visible"))
			Expect(logOutput).NotTo(ContainSubstring("request-secret"))
			Expect(logOutput).NotTo(ContainSubstring("response-secret"))
			Expect(logOutput).NotTo(ContainSubstring("super-secret"))
		})

		It("redacts configured form fields in logs", func() {
			logBuffer := &bytes.Buffer{}
			logger := logrus.New()
			logger.SetOutput(logBuffer)
			logger.SetLevel(logrus.DebugLevel)
			logger.SetFormatter(&logrus.TextFormatter{DisableColors: true, DisableQuote: true})

			roundTripper, err := NewRoundTripper().
				Logger(logger).
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
	})
})
