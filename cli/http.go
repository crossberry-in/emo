package cli

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"runtime"
)

// newHTTPClient creates an *http.Client that uses the system's CA certificate
// pool. If the system cert pool is unavailable (common in minimal containers,
// WSL, or systems without ca-certificates installed), it falls back to
// insecure mode if EMO_INSECURE=1 is set, or returns a helpful error.
//
// Set EMO_INSECURE=1 to skip TLS certificate verification entirely. This is
// NOT recommended for production — install ca-certificates instead:
//
//   Debian/Ubuntu:  apt-get install -y ca-certificates && update-ca-certificates
//   Alpine:         apk add ca-certificates
//   CentOS/RHEL:    yum install ca-certificates && update-ca-trust
//   macOS:          brew install curl-ca-bundle
func newHTTPClient() *http.Client {
	// Try to use the system cert pool.
	transport := &http.Transport{}

	if certPool, err := x509.SystemCertPool(); err == nil && certPool != nil {
		// System cert pool available — use it.
		transport.TLSClientConfig = &tls.Config{
			RootCAs:            certPool,
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: false,
		}
		return &http.Client{Transport: transport}
	}

	// System cert pool not available. Check if user wants insecure mode.
	if os.Getenv("EMO_INSECURE") == "1" {
		fmt.Fprintln(os.Stderr, "⚠️  EMO_INSECURE=1 set — TLS certificate verification disabled.")
		fmt.Fprintln(os.Stderr, "   This is insecure. Install ca-certificates instead:")
		fmt.Fprintln(os.Stderr, "   Debian/Ubuntu: apt-get install -y ca-certificates && update-ca-certificates")
		fmt.Fprintln(os.Stderr, "   Alpine:        apk add ca-certificates")
		fmt.Fprintln(os.Stderr, "   CentOS/RHEL:   yum install ca-certificates && update-ca-trust")
		fmt.Fprintln(os.Stderr)
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
		}
		return &http.Client{Transport: transport}
	}

	// No system certs and no insecure flag — use Go's default pool but
	// give a helpful error if TLS fails. We still try the default transport
	// because Go bundles some CA certs on some platforms.
	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	return &http.Client{Transport: transport}
}

// httpClient is the shared HTTP client for all CLI HTTP requests.
var httpClient = newHTTPClient()

// httpGet is a convenience wrapper that uses the shared HTTP client.
func httpGet(url string) (*http.Response, error) {
	return httpClient.Get(url)
}

// tlsHelpMessage returns a platform-specific help message for fixing TLS
// certificate errors.
func tlsHelpMessage() string {
	switch runtime.GOOS {
	case "linux":
		return `TLS certificate verification failed.

This usually means your system is missing CA certificates. Fix it by running:

  Debian/Ubuntu:  apt-get install -y ca-certificates && update-ca-certificates
  Alpine:         apk add ca-certificates
  CentOS/RHEL:    yum install ca-certificates && update-ca-trust

Or, as a temporary workaround, skip TLS verification:

  export EMO_INSECURE=1

(Not recommended — install ca-certificates instead.)`
	case "darwin":
		return `TLS certificate verification failed.

Install CA certificates:

  brew install curl-ca-bundle

Or, as a temporary workaround:

  export EMO_INSECURE=1`
	default:
		return `TLS certificate verification failed.

Set EMO_INSECURE=1 to skip TLS verification (not recommended).`
	}
}
