package aws

import (
	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net/url"
)

// GetOIDCThumbprint fetches the TLS certificate from the OIDC issuer URL
// and computes the SHA-1 thumbprint of the root certificate.
// This thumbprint is required when creating an AWS IAM OIDC provider.
func GetOIDCThumbprint(issuerURL string) (string, error) {
	// Parse the issuer URL
	u, err := url.Parse(issuerURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse issuer URL: %w", err)
	}

	// Ensure it's HTTPS
	if u.Scheme != "https" {
		return "", fmt.Errorf("issuer URL must use HTTPS scheme, got: %s", u.Scheme)
	}

	// Determine the host and port
	host := u.Host
	if u.Port() == "" {
		host = host + ":443"
	}

	// Connect to the server and get the certificate chain
	conn, err := tls.Dial("tcp", host, &tls.Config{
		// We don't verify the certificate here because we just need to get it
		// to compute the thumbprint
		InsecureSkipVerify: true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to connect to issuer URL: %w", err)
	}
	defer conn.Close()

	// Get the certificate chain
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return "", fmt.Errorf("no certificates found in the chain")
	}

	// Get the root certificate (last in the chain)
	// AWS requires the thumbprint of the top intermediate CA or root CA
	rootCert := certs[len(certs)-1]

	// Compute SHA-1 hash of the DER-encoded certificate
	hash := sha1.Sum(rootCert.Raw)
	thumbprint := hex.EncodeToString(hash[:])

	return thumbprint, nil
}
