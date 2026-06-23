package clients

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestClients(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Clients Suite")
}

var _ = Describe("DefaultHTTPClient", func() {
	It("issues a GET request with the wrapped client", func() {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			Expect(request.Method).To(Equal(http.MethodGet))
			_, err := writer.Write([]byte("ok"))
			Expect(err).NotTo(HaveOccurred())
		}))
		defer server.Close()

		client := NewDefaultHTTPClient(server.Client())

		response, err := client.Get(server.URL)
		Expect(err).NotTo(HaveOccurred())
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(body)).To(Equal("ok"))
	})

	It("returns the underlying client error", func() {
		client := NewDefaultHTTPClient(http.DefaultClient)

		_, err := client.Get("://bad-url")
		Expect(err).To(HaveOccurred())
	})
})
