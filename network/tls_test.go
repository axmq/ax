package network

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultTLSConfig(t *testing.T) {
	config := DefaultTLSConfig()
	assert.NotNil(t, config)
	assert.Equal(t, tls.NoClientCert, config.ClientAuth)
	assert.Equal(t, uint16(tls.VersionTLS13), config.MinVersion)
}

func TestTLSConfigBuildMissingCert(t *testing.T) {
	config := &TLSConfig{
		CertFile: "",
		KeyFile:  "",
	}
	_, err := config.Build()
	assert.Equal(t, ErrInvalidTLSConfig, err)
}

func TestTLSConfigBuildMissingKey(t *testing.T) {
	config := &TLSConfig{
		CertFile: "cert.pem",
		KeyFile:  "",
	}
	_, err := config.Build()
	assert.Equal(t, ErrInvalidTLSConfig, err)
}

func TestGetPeerCertificates(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)

	certs, err := GetPeerCertificates(conn)
	assert.NoError(t, err)
	assert.Nil(t, certs)
}

func TestGetPeerCommonName(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)

	cn, err := GetPeerCommonName(conn)
	assert.NoError(t, err)
	assert.Empty(t, cn)
}

func TestVerifyPeerCertificateNonTLS(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := NewConnection(server, "test-conn", nil)

	err := VerifyPeerCertificate(conn, "expected-cn")
	assert.NoError(t, err)
}

func TestNewTLSVerifierEmptyCA(t *testing.T) {
	verifier, err := NewTLSVerifier("")
	assert.NoError(t, err)
	assert.NotNil(t, verifier)
}

func TestNewTLSVerifierInvalidFile(t *testing.T) {
	verifier, err := NewTLSVerifier("/nonexistent/ca.pem")
	assert.Error(t, err)
	assert.Nil(t, verifier)
}

func TestTLSVerifierSetCustomVerifier(t *testing.T) {
	verifier, err := NewTLSVerifier("")
	require.NoError(t, err)

	customCalled := false
	verifier.SetCustomVerifier(func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		customCalled = true
		return nil
	})

	err = verifier.VerifyCertificate([][]byte{}, nil)
	assert.NoError(t, err)
	assert.True(t, customCalled)
}

func TestTLSVerifierVerifyCertificateNoCerts(t *testing.T) {
	verifier, err := NewTLSVerifier("")
	require.NoError(t, err)

	err = verifier.VerifyCertificate([][]byte{}, nil)
	assert.Equal(t, ErrCertificateVerification, err)
}

func TestMutualTLSConfigBuild(t *testing.T) {
	mtc := &MutualTLSConfig{
		TLSConfig: TLSConfig{
			CertFile: "",
			KeyFile:  "",
		},
		RequireClientCert: true,
		VerifyClientCert:  true,
	}

	_, err := mtc.Build()
	assert.Error(t, err)
}

func TestDefaultTLSConfigValues(t *testing.T) {
	config := DefaultTLSConfig()
	assert.Equal(t, tls.NoClientCert, config.ClientAuth)
	assert.Equal(t, uint16(tls.VersionTLS13), config.MinVersion)
	assert.Equal(t, uint16(tls.VersionTLS13), config.MaxVersion)
	assert.False(t, config.InsecureSkipVerify)
	assert.Empty(t, config.CipherSuites)
}

func TestTLSConfigBuildWithValidCerts(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")

	certPEM := []byte(`-----BEGIN CERTIFICATE-----
MIIB9TCCAV6gAwIBAgIRAIuPYZlKy5cMPLKMwXPLAH8wDQYJKoZIhvcNAQELBQAw
ETEPMA0GA1UEAxMGdGVzdENBMB4XDTI0MDEwMTAwMDAwMFoXDTM0MDEwMTAwMDAw
MFowETEPMA0GA1UEAxMGdGVzdENBMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKB
gQC6Z8V7DP/1vRlGg8K8rk9lczP+s8jQ6kJvH6kzDW3VB6y6sQkdQxY1shKQPwAb
JK+WYPGpKnxDMdIBPx6Zi5Q3l7RgxMgMzqW3eU7HqHF0t2OwYsVHPGxF3P3OEQdm
KMhvCPj4AqwsH0mVnJ2v4nBUGP7vbqQVvLLdSLDNpJKwmQIDAQABo0UwQzAOBgNV
HQ8BAf8EBAMCAqQwEwYDVR0lBAwwCgYIKwYBBQUHAwEwDwYDVR0TAQH/BAUwAwEB
/zALBgNVHQ8EBAMCAoQwDQYJKoZIhvcNAQELBQADgYEADvJ6V8ycJQY1mCL0Yd6o
vPRYQ3Vqx5LKQfUGBdmGTGWP/bJQvJvNrGDSQVCvbqT5Y9Ky3D1FNQpCPgQKx0Nw
zXBmKkCl2LJPhHUVqQ7nF8HqVlKg0v2DvxHRgqANPCxGZvJnKb4pLPKCNkQdIAFV
2y0kcKkBQx/lPJ5RMZhUJWE=
-----END CERTIFICATE-----`)

	keyPEM := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIICXAIBAAKBgQC6Z8V7DP/1vRlGg8K8rk9lczP+s8jQ6kJvH6kzDW3VB6y6sQkd
QxY1shKQPwAbJK+WYPGpKnxDMdIBPx6Zi5Q3l7RgxMgMzqW3eU7HqHF0t2OwYsVH
PGXF3P3OEQdmKMhvCPj4AqwsH0mVnJ2v4nBUGP7vbqQVvLLdSLDNpJKwmQIDAQAB
AoGAX8E2T2J8vYPQZvPJvF9F7GqJQEWzMvw3cQzEYXNMQJYLnBmP7B0jMPIVvCxE
CvE5HRgQQCvNJT2FzHNvJ0kLwYQh/C6TBgvYMZQrp0xPNJWKGCYC0cLPvZLBB7E6
F0qVqVHHHqvXvxYQz0LxP7tFhLbxoJQBPZLQx6K+F6+nLAECQQDiT7Lyl3BYvR2M
LZ0F6vNYLCqMv3f0E8fYFvL3CQx3vT7vWvL6dQFmLX6F7LLVvLL4vWYFhQdVPbL5
vxJYz0yRAkEA0rL7xf3Q5JW6P7Q9YFv4LCvQVzL8vPYLF3mBvQx3YFLxW3xLF6vL
xQxQvLvYPxFvBYFxL3L7YxPvLxYFxL5QmQJAX3J0PYxFLvY3FxLYQx7vLxYFLxL3
QYFxL6PxQYL3vxFxL3YxPLvxYFxL3LYQxPLvxYFLxL3QYFxL6PxQYL3vxECQHxL3
YxPLvxYFxL3LYQxPLvxYFLxL3QYFxL6PxQYL3vxFxL3YxPLvxYFxL3LYQxPLvxYF
LxL3QYFxL6PxQYL3vxECQQC5L3YxPLvxYFxL3LYQxPLvxYFLxL3QYFxL6PxQYL3v
xFxL3YxPLvxYFxL3LYQxPLvxYFLxL3QYFxL6PxQYL3vx
-----END RSA PRIVATE KEY-----`)

	err := os.WriteFile(certFile, certPEM, 0600)
	require.NoError(t, err)
	err = os.WriteFile(keyFile, keyPEM, 0600)
	require.NoError(t, err)

	config := &TLSConfig{
		CertFile:   certFile,
		KeyFile:    keyFile,
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
	}

	tlsConfig, err := config.Build()
	require.NoError(t, err)
	assert.NotNil(t, tlsConfig)
	assert.Equal(t, uint16(tls.VersionTLS12), tlsConfig.MinVersion)
	assert.Equal(t, uint16(tls.VersionTLS13), tlsConfig.MaxVersion)
}

func TestTLSConfigBuildWithCA(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")
	caFile := filepath.Join(tmpDir, "ca.pem")

	certPEM := []byte(`-----BEGIN CERTIFICATE-----
MIIB9TCCAV6gAwIBAgIRAIuPYZlKy5cMPLKMwXPLAH8wDQYJKoZIhvcNAQELBQAw
ETEPMA0GA1UEAxMGdGVzdENBMB4XDTI0MDEwMTAwMDAwMFoXDTM0MDEwMTAwMDAw
MFowETEPMA0GA1UEAxMGdGVzdENBMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKB
gQC6Z8V7DP/1vRlGg8K8rk9lczP+s8jQ6kJvH6kzDW3VB6y6sQkdQxY1shKQPwAb
JK+WYPGpKnxDMdIBPx6Zi5Q3l7RgxMgMzqW3eU7HqHF0t2OwYsVHPGxF3P3OEQdm
KMhvCPj4AqwsH0mVnJ2v4nBUGP7vbqQVvLLdSLDNpJKwmQIDAQABo0UwQzAOBgNV
HQ8BAf8EBAMCAqQwEwYDVR0lBAwwCgYIKwYBBQUHAwEwDwYDVR0TAQH/BAUwAwEB
/zALBgNVHQ8EBAMCAoQwDQYJKoZIhvcNAQELBQADgYEADvJ6V8ycJQY1mCL0Yd6o
vPRYQ3Vqx5LKQfUGBdmGTGWP/bJQvJvNrGDSQVCvbqT5Y9Ky3D1FNQpCPgQKx0Nw
zXBmKkCl2LJPhHUVqQ7nF8HqVlKg0v2DvxHRgqANPCxGZvJnKb4pLPKCNkQdIAFV
2y0kcKkBQx/lPJ5RMZhUJWE=
-----END CERTIFICATE-----`)

	keyPEM := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIICXAIBAAKBgQC6Z8V7DP/1vRlGg8K8rk9lczP+s8jQ6kJvH6kzDW3VB6y6sQkd
QxY1shKQPwAbJK+WYPGpKnxDMdIBPx6Zi5Q3l7RgxMgMzqW3eU7HqHF0t2OwYsVH
PGXF3P3OEQdmKMhvCPj4AqwsH0mVnJ2v4nBUGP7vbqQVvLLdSLDNpJKwmQIDAQAB
AoGAX8E2T2J8vYPQZvPJvF9F7GqJQEWzMvw3cQzEYXNMQJYLnBmP7B0jMPIVvCxE
CvE5HRgQQCvNJT2FzHNvJ0kLwYQh/C6TBgvYMZQrp0xPNJWKGCYC0cLPvZLBB7E6
F0qVqVHHHqvXvxYQz0LxP7tFhLbxoJQBPZLQx6K+F6+nLAECQQDiT7Lyl3BYvR2M
LZ0F6vNYLCqMv3f0E8fYFvL3CQx3vT7vWvL6dQFmLX6F7LLVvLL4vWYFhQdVPbL5
vxJYz0yRAkEA0rL7xf3Q5JW6P7Q9YFv4LCvQVzL8vPYLF3mBvQx3YFLxW3xLF6vL
xQxQvLvYPxFvBYFxL3L7YxPvLxYFxL5QmQJAX3J0PYxFLvY3FxLYQx7vLxYFLxL3
QYFxL6PxQYL3vxFxL3YxPLvxYFxL3LYQxPLvxYFLxL3QYFxL6PxQYL3vxECQHxL3
YxPLvxYFxL3LYQxPLvxYFLxL3QYFxL6PxQYL3vxFxL3YxPLvxYFxL3LYQxPLvxYF
LxL3QYFxL6PxQYL3vxECQQC5L3YxPLvxYFxL3LYQxPLvxYFLxL3QYFxL6PxQYL3v
xFxL3YxPLvxYFxL3LYQxPLvxYFLxL3QYFxL6PxQYL3vx
-----END RSA PRIVATE KEY-----`)

	var err error
	err = os.WriteFile(certFile, certPEM, 0600)
	require.NoError(t, err)
	err = os.WriteFile(keyFile, keyPEM, 0600)
	require.NoError(t, err)
	err = os.WriteFile(caFile, certPEM, 0600)
	require.NoError(t, err)

	config := &TLSConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
	}

	tlsConfig, err := config.Build()
	require.NoError(t, err)
	assert.NotNil(t, tlsConfig)
	assert.NotNil(t, tlsConfig.ClientCAs)
	assert.Equal(t, tls.RequireAndVerifyClientCert, tlsConfig.ClientAuth)
}

func TestTLSConfigBuildInvalidCertFile(t *testing.T) {
	config := &TLSConfig{
		CertFile: "/nonexistent/cert.pem",
		KeyFile:  "/nonexistent/key.pem",
	}

	_, err := config.Build()
	assert.Error(t, err)
}

func TestTLSConfigBuildInvalidCAFile(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")
	caFile := filepath.Join(tmpDir, "ca.pem")

	certPEM := []byte(`-----BEGIN CERTIFICATE-----
MIIB9TCCAV6gAwIBAgIRAIuPYZlKy5cMPLKMwXPLAH8wDQYJKoZIhvcNAQELBQAw
ETEPMA0GA1UEAxMGdGVzdENBMB4XDTI0MDEwMTAwMDAwMFoXDTM0MDEwMTAwMDAw
MFowETEPMA0GA1UEAxMGdGVzdENBMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKB
gQC6Z8V7DP/1vRlGg8K8rk9lczP+s8jQ6kJvH6kzDW3VB6y6sQkdQxY1shKQPwAb
JK+WYPGpKnxDMdIBPx6Zi5Q3l7RgxMgMzqW3eU7HqHF0t2OwYsVHPGxF3P3OEQdm
KMhvCPj4AqwsH0mVnJ2v4nBUGP7vbqQVvLLdSLDNpJKwmQIDAQABo0UwQzAOBgNV
HQ8BAf8EBAMCAqQwEwYDVR0lBAwwCgYIKwYBBQUHAwEwDwYDVR0TAQH/BAUwAwEB
/zALBgNVHQ8EBAMCAoQwDQYJKoZIhvcNAQELBQADgYEADvJ6V8ycJQY1mCL0Yd6o
vPRYQ3Vqx5LKQfUGBdmGTGWP/bJQvJvNrGDSQVCvbqT5Y9Ky3D1FNQpCPgQKx0Nw
zXBmKkCl2LJPhHUVqQ7nF8HqVlKg0v2DvxHRgqANPCxGZvJnKb4pLPKCNkQdIAFV
2y0kcKkBQx/lPJ5RMZhUJWE=
-----END CERTIFICATE-----`)

	keyPEM := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIICXAIBAAKBgQC6Z8V7DP/1vRlGg8K8rk9lczP+s8jQ6kJvH6kzDW3VB6y6sQkd
QxY1shKQPwAbJK+WYPGpKnxDMdIBPx6Zi5Q3l7RgxMgMzqW3eU7HqHF0t2OwYsVH
PGXF3P3OEQdmKMhvCPj4AqwsH0mVnJ2v4nBUGP7vbqQVvLLdSLDNpJKwmQIDAQAB
AoGAX8E2T2J8vYPQZvPJvF9F7GqJQEWzMvw3cQzEYXNMQJYLnBmP7B0jMPIVvCxE
CvE5HRgQQCvNJT2FzHNvJ0kLwYQh/C6TBgvYMZQrp0xPNJWKGCYC0cLPvZLBB7E6
F0qVqVHHHqvXvxYQz0LxP7tFhLbxoJQBPZLQx6K+F6+nLAECQQDiT7Lyl3BYvR2M
LZ0F6vNYLCqMv3f0E8fYFvL3CQx3vT7vWvL6dQFmLX6F7LLVvLL4vWYFhQdVPbL5
vxJYz0yRAkEA0rL7xf3Q5JW6P7Q9YFv4LCvQVzL8vPYLF3mBvQx3YFLxW3xLF6vL
xQxQvLvYPxFvBYFxL3L7YxPvLxYFxL5QmQJAX3J0PYxFLvY3FxLYQx7vLxYFLxL3
QYFxL6PxQYL3vxFxL3YxPLvxYFxL3LYQxPLvxYFLxL3QYFxL6PxQYL3vxECQHxL3
YxPLvxYFxL3LYQxPLvxYFLxL3QYFxL6PxQYL3vxFxL3YxPLvxYFxL3LYQxPLvxYF
LxL3QYFxL6PxQYL3vxECQQC5L3YxPLvxYFxL3LYQxPLvxYFLxL3QYFxL6PxQYL3v
xFxL3YxPLvxYFxL3LYQxPLvxYFLxL3QYFxL6PxQYL3vx
-----END RSA PRIVATE KEY-----`)

	var err error
	err = os.WriteFile(certFile, certPEM, 0600)
	require.NoError(t, err)
	err = os.WriteFile(keyFile, keyPEM, 0600)
	require.NoError(t, err)
	err = os.WriteFile(caFile, certPEM, 0600)
	require.NoError(t, err)

	config := &TLSConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   "/nonexistent/ca.pem",
	}

	_, err = config.Build()
	assert.Error(t, err)
}
