/*
Copyright (c) 2026 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package login

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/authentication"

	"github.com/openshift/rosa/pkg/config"
	"github.com/openshift/rosa/pkg/fedramp"
	"github.com/openshift/rosa/pkg/rosa"
)

func TestMain(m *testing.M) {
	if os.Getenv("ROSA_BROWSER_OPENER_HELPER") == "1" {
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func signLoginToken(tokenType string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp":      time.Now().Add(time.Hour).Unix(),
		"typ":      tokenType,
		"username": "test-user",
	})
	return token.SignedString([]byte("test-signing-key"))
}

func makeLoginToken(tokenType string) string {
	value, err := signLoginToken(tokenType)
	Expect(err).ToNot(HaveOccurred())
	return value
}

func resetLoginArgs() {
	args.tokenURL = ""
	args.clientID = ""
	args.clientSecret = ""
	args.scopes = append([]string(nil), sdk.DefaultScopes...)
	args.env = ""
	args.token = ""
	args.insecure = false
	args.useAuthCode = false
	args.useDeviceCode = false
	args.rhRegion = ""
}

func makeAuthCodeProxyCertificate(t *testing.T) ([]byte, tls.Certificate) {
	t.Helper()
	now := time.Now()
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "ROSA auth-code test CA"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "sso.redhat.com"},
		DNSNames:     []string{"sso.redhat.com"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	serverDER, err := x509.CreateCertificate(
		rand.Reader, serverTemplate, caTemplate, &serverKey.PublicKey, caKey,
	)
	if err != nil {
		t.Fatal(err)
	}
	certificatePEM := append(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER}),
		caPEM...,
	)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey)})
	serverCertificate, err := tls.X509KeyPair(certificatePEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	return caPEM, serverCertificate
}

func makeCertPool(t *testing.T, certificatePEM []byte) *x509.CertPool {
	t.Helper()
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(certificatePEM) {
		t.Fatal("failed to add certificate to trusted CA pool")
	}
	return pool
}

type browserCallbackResult struct {
	connected  bool
	statusCode int
	err        error
}

func runBrowserCallback(ctx context.Context) browserCallbackResult {
	var connected atomic.Bool
	dialer := &net.Dialer{}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			connection, err := dialer.DialContext(ctx, network, address)
			if err == nil {
				connected.Store(true)
			}
			return connection, err
		},
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}
	callbackURL := fmt.Sprintf(
		"http://127.0.0.1:%s%s?code=test-code",
		authentication.RedirectPort,
		authentication.CallbackHandler,
	)

	for {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, callbackURL, nil)
		if err != nil {
			return browserCallbackResult{err: err}
		}
		response, err := client.Do(request) // #nosec G107 -- fixed loopback test callback
		if connected.Load() {
			result := browserCallbackResult{connected: true, err: err}
			if response != nil {
				result.statusCode = response.StatusCode
				_ = response.Body.Close()
				if response.StatusCode != http.StatusOK {
					result.err = fmt.Errorf("auth-code callback returned status %d", response.StatusCode)
				}
			}
			return result
		}
		if ctx.Err() != nil {
			return browserCallbackResult{err: ctx.Err()}
		}
		if !errors.Is(err, syscall.ECONNREFUSED) {
			return browserCallbackResult{err: err}
		}

		timer := time.NewTimer(20 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return browserCallbackResult{err: ctx.Err()}
		case <-timer.C:
		}
	}
}

func callbackPortAvailable() error {
	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", authentication.RedirectPort))
	if err != nil {
		return err
	}
	return listener.Close()
}

func TestBrowserCallbackCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := runBrowserCallback(ctx)
	if result.connected {
		t.Fatal("canceled callback unexpectedly established a connection")
	}
	if !errors.Is(result.err, context.Canceled) {
		t.Fatalf("expected canceled callback, got %v", result.err)
	}
}

func copyTestBinary(source string, destination string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0700)
	if err != nil {
		return err
	}
	if _, err := io.Copy(destinationFile, sourceFile); err != nil {
		_ = destinationFile.Close()
		return err
	}
	return destinationFile.Close()
}

func makeControlledBrowserOpener(directory string) (string, error) {
	testBinary, err := os.Executable()
	if err != nil {
		return "", err
	}

	var (
		openerDirectory string
		openerName      string
		environment     string
	)
	switch runtime.GOOS {
	case "darwin":
		openerDirectory = filepath.Join(directory, "bin")
		openerName = "open"
		environment = "PATH=" + openerDirectory + string(os.PathListSeparator) + os.Getenv("PATH")
	case "windows":
		// open-golang resolves rundll32.exe from SYSTEMROOT during package initialization,
		// so the auth-code helper subprocess must start with this isolated root.
		systemRoot := filepath.Join(directory, "system-root")
		openerDirectory = filepath.Join(systemRoot, "System32")
		openerName = "rundll32.exe"
		environment = "SYSTEMROOT=" + systemRoot
	default:
		openerDirectory = filepath.Join(directory, "bin")
		openerName = "xdg-open"
		environment = "PATH=" + openerDirectory + string(os.PathListSeparator) + os.Getenv("PATH")
	}

	if err := os.MkdirAll(openerDirectory, 0700); err != nil {
		return "", err
	}
	if err := copyTestBinary(testBinary, filepath.Join(openerDirectory, openerName)); err != nil {
		return "", err
	}
	return environment, nil
}

func environmentWithout(names ...string) []string {
	excluded := make(map[string]struct{}, len(names))
	for _, name := range names {
		excluded[strings.ToUpper(name)] = struct{}{}
	}
	environment := make([]string, 0, len(os.Environ()))
	for _, value := range os.Environ() {
		name, _, _ := strings.Cut(value, "=")
		if _, ok := excluded[strings.ToUpper(name)]; !ok {
			environment = append(environment, value)
		}
	}
	return environment
}

func supportsSSLCertFile(goos string) bool {
	// Keep this aligned with crypto/x509/root_unix.go. Android and illumos
	// also match the linux and solaris build tags, respectively.
	switch goos {
	case "aix", "android", "dragonfly", "freebsd", "illumos", "js", "linux", "netbsd", "openbsd", "solaris", "wasip1":
		return true
	default:
		return false
	}
}

func TestSupportsSSLCertFile(t *testing.T) {
	for _, goos := range []string{
		"aix", "android", "dragonfly", "freebsd", "illumos", "js", "linux", "netbsd", "openbsd", "solaris", "wasip1",
	} {
		if !supportsSSLCertFile(goos) {
			t.Errorf("expected SSL_CERT_FILE support on %s", goos)
		}
	}
	for _, goos := range []string{"darwin", "ios", "plan9", "windows"} {
		if supportsSSLCertFile(goos) {
			t.Errorf("did not expect SSL_CERT_FILE support on %s", goos)
		}
	}
}

func preserveLoginTestState() func() error {
	originalContext := Cmd.Context()
	originalOCMConfig, originalOCMConfigSet := os.LookupEnv("OCM_CONFIG")
	return func() error {
		Cmd.SetContext(originalContext)
		if originalOCMConfigSet {
			return os.Setenv("OCM_CONFIG", originalOCMConfig)
		}
		return os.Unsetenv("OCM_CONFIG")
	}
}

func TestAuthCodeTLSHelper(t *testing.T) {
	scenario := os.Getenv("ROSA_AUTH_CODE_TLS_HELPER")
	if scenario == "" {
		t.Skip("helper process")
	}
	if scenario != "untrusted" && scenario != "insecure" && scenario != "trusted" {
		t.Fatalf("unknown TLS scenario %q", scenario)
	}

	caPEM, serverCertificate := makeAuthCodeProxyCertificate(t)
	accessToken, err := signLoginToken("Bearer")
	if err != nil {
		t.Fatal(err)
	}
	refreshToken, err := signLoginToken("Refresh")
	if err != nil {
		t.Fatal(err)
	}
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		connection, _, err := hijacker.Hijack()
		if err != nil {
			return
		}
		defer connection.Close()
		_, _ = io.WriteString(connection, "HTTP/1.1 200 Connection Established\r\n\r\n")
		tlsConnection := tls.Server(connection, &tls.Config{
			Certificates: []tls.Certificate{serverCertificate},
			MinVersion:   tls.VersionTLS12,
		})
		if err := tlsConnection.Handshake(); err != nil {
			return
		}
		request, err := http.ReadRequest(bufio.NewReader(tlsConnection))
		if err != nil {
			return
		}
		if request.Body != nil {
			_, _ = io.Copy(io.Discard, request.Body)
			_ = request.Body.Close()
		}
		body := `{"access_token":"` + accessToken + `","refresh_token":"` + refreshToken +
			`","token_type":"Bearer","expires_in":3600}`
		_, _ = fmt.Fprintf(
			tlsConnection,
			"HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s",
			len(body), body,
		)
	}))
	defer proxy.Close()

	t.Setenv("HTTPS_PROXY", proxy.URL)
	t.Setenv("https_proxy", proxy.URL)
	t.Setenv("NO_PROXY", "")
	t.Setenv("no_proxy", "")
	_ = os.Unsetenv("SSL_CERT_FILE")

	// The opener is a copy of this test binary prepared by the parent spec. TestMain
	// exits it immediately so no supported platform can launch a real browser. The
	// parent spec owns, cancels, and joins the callback request separately.
	t.Setenv("ROSA_BROWSER_OPENER_HELPER", "1")

	authCodeConfig := authentication.NewAuthCodeConfig().
		Client(oauthClientId).
		Insecure(scenario == "insecure")
	if scenario == "trusted" {
		authCodeConfig.TrustedCA(makeCertPool(t, caPEM))
	}
	token, err := authCodeConfig.InitiateAuthCode(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if token != refreshToken {
		t.Fatalf("expected refresh token from auth-code exchange")
	}
}

func TestOCMTrustedCAHelper(t *testing.T) {
	if os.Getenv("ROSA_OCM_TRUSTED_CA_HELPER") != "1" {
		t.Skip("helper process")
	}

	accessToken, err := signLoginToken("Bearer")
	if err != nil {
		t.Fatal(err)
	}
	refreshToken, err := signLoginToken("Refresh")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"` + accessToken + `"}`))
	}))
	defer server.Close()
	certificate := server.Certificate()
	if certificate == nil {
		t.Fatal("TLS server certificate is missing")
	}
	trustedCAs := makeCertPool(
		t,
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate.Raw}),
	)
	connection, err := sdk.NewConnectionBuilder().
		URL(server.URL).
		TokenURL(server.URL).
		Tokens(refreshToken).
		TrustedCAs(trustedCAs).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if connection.Insecure() {
		t.Fatal("trusted OCM connection unexpectedly disabled TLS verification")
	}
	actualAccessToken, actualRefreshToken, err := connection.Tokens()
	if err != nil {
		t.Fatal(err)
	}
	if actualAccessToken != accessToken || actualRefreshToken != refreshToken {
		t.Fatal("trusted OCM connection returned unexpected tokens")
	}
}

func TestLoginSystemTrustHelper(t *testing.T) {
	if os.Getenv("ROSA_LOGIN_SYSTEM_TRUST_HELPER") != "1" {
		t.Skip("helper process")
	}

	resetLoginArgs()
	reAttempt = false
	env = ""
	fedramp.Disable()
	accessToken, err := signLoginToken("Bearer")
	if err != nil {
		t.Fatal(err)
	}
	refreshToken, err := signLoginToken("Refresh")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"` + accessToken + `"}`))
	}))
	defer server.Close()
	certificate := server.Certificate()
	if certificate == nil {
		t.Fatal("TLS server certificate is missing")
	}
	caFile := filepath.Join(t.TempDir(), "trusted-ca.pem")
	caData := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate.Raw})
	if caData == nil {
		t.Fatal("failed to encode TLS server certificate")
	}
	if err := os.WriteFile(caFile, caData, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SSL_CERT_FILE", caFile)
	t.Setenv("OCM_CONFIG", filepath.Join(t.TempDir(), "ocm-config.json"))
	initiateAuthCode = func(ctx context.Context, clientID string, insecure bool) (string, error) {
		if ctx == nil || clientID != oauthClientId || insecure {
			return "", errors.New("unexpected auth-code options")
		}
		return refreshToken, nil
	}
	args.useAuthCode = true
	args.tokenURL = server.URL
	args.env = server.URL
	Cmd.SetContext(context.Background())
	runtime := rosa.NewRuntime()
	defer runtime.Cleanup()
	if err := runWithRuntime(runtime, Cmd, nil); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ClientID != oauthClientId || cfg.AccessToken != accessToken ||
		cfg.RefreshToken != refreshToken || cfg.Insecure {
		t.Fatalf("unexpected persisted login configuration: client=%q insecure=%t", cfg.ClientID, cfg.Insecure)
	}
}

var _ = Describe("OAuth authorization code TLS options", Ordered, func() {
	var (
		originalInitiateAuthCode = initiateAuthCode
		untrustedServer          *httptest.Server
		accessToken              string
		refreshToken             string
	)

	BeforeAll(func() {
		untrustedServer = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"` + accessToken + `"}`))
		}))
		accessToken = makeLoginToken("Bearer")
		refreshToken = makeLoginToken("Refresh")
	})

	AfterAll(func() {
		untrustedServer.Close()
		resetLoginArgs()
		env = ""
		reAttempt = false
		initiateAuthCode = originalInitiateAuthCode
	})

	BeforeEach(func() {
		restoreState := preserveLoginTestState()
		DeferCleanup(func() {
			Expect(restoreState()).To(Succeed())
		})

		resetLoginArgs()
		env = ""
		reAttempt = false
		fedramp.Disable()
		Cmd.SetContext(context.Background())
		Expect(os.Setenv("OCM_CONFIG", filepath.Join(GinkgoT().TempDir(), "ocm-config.json"))).To(Succeed())
		initiateAuthCode = func(ctx context.Context, clientID string, insecure bool) (string, error) {
			Expect(ctx).ToNot(BeNil())
			Expect(clientID).To(Equal(oauthClientId))
			Expect(insecure).To(Equal(args.insecure))
			return refreshToken, nil
		}
	})

	runLogin := func(serverURL string, insecure bool) error {
		args.useAuthCode = true
		args.insecure = insecure
		args.tokenURL = serverURL
		args.env = serverURL
		runtime := rosa.NewRuntime()
		defer runtime.Cleanup()
		return runWithRuntime(runtime, Cmd, nil)
	}

	It("enforces TLS verification during the auth-code token exchange", func() {
		openerEnvironment, err := makeControlledBrowserOpener(GinkgoT().TempDir())
		Expect(err).ToNot(HaveOccurred())
		openerEnvironmentName, _, _ := strings.Cut(openerEnvironment, "=")
		for _, scenario := range []string{"untrusted", "insecure", "trusted"} {
			Expect(callbackPortAvailable()).To(Succeed(), "callback port was contaminated before %s", scenario)
			scenarioContext, cancelScenario := context.WithTimeout(context.Background(), 15*time.Second)
			callbackResults := make(chan browserCallbackResult, 1)
			go func() {
				callbackResults <- runBrowserCallback(scenarioContext)
			}()

			command := exec.CommandContext(scenarioContext, os.Args[0], "-test.run=^TestAuthCodeTLSHelper$")
			environment := environmentWithout(
				"SSL_CERT_FILE",
				"ROSA_AUTH_CODE_TLS_HELPER",
				"ROSA_BROWSER_CALLBACK_HELPER",
				"ROSA_BROWSER_OPENER_HELPER",
				openerEnvironmentName,
			)
			command.Env = append(
				environment,
				openerEnvironment,
				"ROSA_AUTH_CODE_TLS_HELPER="+scenario,
			)
			output, err := command.CombinedOutput()
			cancelScenario()
			callbackResult := <-callbackResults
			Expect(callbackResult.connected).To(BeTrue(), "%s callback did not reach the SDK server: %v", scenario, callbackResult.err)
			if scenario == "untrusted" {
				Expect(err).To(HaveOccurred(), string(output))
				Expect(string(output)).To(Or(ContainSubstring("certificate"), ContainSubstring("x509")))
			} else {
				Expect(err).ToNot(HaveOccurred(), "%s scenario failed:\n%s", scenario, string(output))
				Expect(callbackResult.err).ToNot(HaveOccurred())
				Expect(callbackResult.statusCode).To(Equal(http.StatusOK))
			}
			Eventually(callbackPortAvailable, time.Second, 10*time.Millisecond).Should(
				Succeed(), "callback port remained occupied after %s", scenario,
			)
		}
	})

	It("uses an explicit trusted CA pool for the later OCM connection", func() {
		command := exec.Command(os.Args[0], "-test.run=^TestOCMTrustedCAHelper$")
		environment := environmentWithout("SSL_CERT_FILE", "ROSA_OCM_TRUSTED_CA_HELPER")
		command.Env = append(environment, "ROSA_OCM_TRUSTED_CA_HELPER=1")
		output, err := command.CombinedOutput()
		Expect(err).ToNot(HaveOccurred(), string(output))
	})

	It("uses the system trust store for the full later OCM connection on supported Unix platforms", func() {
		if !supportsSSLCertFile(runtime.GOOS) {
			Skip("Go does not support SSL_CERT_FILE for its system trust pool on this platform")
		}
		command := exec.Command(os.Args[0], "-test.run=^TestLoginSystemTrustHelper$")
		environment := environmentWithout("SSL_CERT_FILE", "ROSA_LOGIN_SYSTEM_TRUST_HELPER")
		command.Env = append(environment, "ROSA_LOGIN_SYSTEM_TRUST_HELPER=1")
		output, err := command.CombinedOutput()
		Expect(err).ToNot(HaveOccurred(), string(output))
	})

	It("rejects an untrusted certificate by default for the later OCM connection", func() {
		err := runLogin(untrustedServer.URL, false)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to create OCM connection"))
		Expect(err.Error()).To(Or(ContainSubstring("certificate"), ContainSubstring("x509")))
	})

	It("accepts an untrusted certificate only when insecure is explicit", func() {
		err := runLogin(untrustedServer.URL, true)
		Expect(err).ToNot(HaveOccurred())

		cfg, err := config.Load()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.Insecure).To(BeTrue())
	})

	It("forwards the context, client ID, and insecure option and returns auth-code errors", func() {
		expectedError := errors.New("auth-code exchange failed")
		ctx := context.WithValue(context.Background(), struct{}{}, "test-context")
		Cmd.SetContext(ctx)
		args.useAuthCode = true
		args.insecure = true
		initiateAuthCode = func(actualContext context.Context, clientID string, insecure bool) (string, error) {
			Expect(actualContext).To(Equal(ctx))
			Expect(clientID).To(Equal(oauthClientId))
			Expect(insecure).To(BeTrue())
			return "", expectedError
		}

		runtime := rosa.NewRuntime()
		defer runtime.Cleanup()
		err := runWithRuntime(runtime, Cmd, nil)
		Expect(err).To(MatchError(ContainSubstring(expectedError.Error())))
	})

	It("restores the exact command context and OCM_CONFIG state", func() {
		for _, testCase := range []struct {
			name  string
			value string
			isSet bool
		}{
			{name: "unset", isSet: false},
			{name: "set empty", value: "", isSet: true},
			{name: "set non-empty", value: "original-config", isSet: true},
		} {
			By(testCase.name)
			if testCase.isSet {
				Expect(os.Setenv("OCM_CONFIG", testCase.value)).To(Succeed())
			} else {
				Expect(os.Unsetenv("OCM_CONFIG")).To(Succeed())
			}
			expectedContext := context.WithValue(context.Background(), struct{}{}, testCase.name)
			Cmd.SetContext(expectedContext)
			restoreState := preserveLoginTestState()

			Cmd.SetContext(context.Background())
			Expect(os.Setenv("OCM_CONFIG", "temporary-config")).To(Succeed())
			Expect(restoreState()).To(Succeed())

			Expect(Cmd.Context() == expectedContext).To(BeTrue())
			actualValue, actualIsSet := os.LookupEnv("OCM_CONFIG")
			Expect(actualIsSet).To(Equal(testCase.isSet))
			if testCase.isSet {
				Expect(actualValue).To(Equal(testCase.value))
			}
		}
	})
})
