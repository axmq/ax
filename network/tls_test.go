package network

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestCertificate creates a valid self-signed certificate for testing
func generateTestCertificate() (certPEM, keyPEM []byte, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "localhost",
		},
		NotBefore:             time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:              time.Date(2034, 1, 1, 0, 0, 0, 0, time.UTC),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, err
	}

	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return certPEM, keyPEM, nil
}

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

	certPEM, keyPEM, err := generateTestCertificate()
	require.NoError(t, err)

	err = os.WriteFile(certFile, certPEM, 0o600)
	require.NoError(t, err)
	err = os.WriteFile(keyFile, keyPEM, 0o600)
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

	certPEM, keyPEM, err := generateTestCertificate()
	require.NoError(t, err)

	err = os.WriteFile(certFile, certPEM, 0o600)
	require.NoError(t, err)
	err = os.WriteFile(keyFile, keyPEM, 0o600)
	require.NoError(t, err)
	err = os.WriteFile(caFile, certPEM, 0o600)
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

	certPEM, keyPEM, err := generateTestCertificate()
	require.NoError(t, err)

	err = os.WriteFile(certFile, certPEM, 0o600)
	require.NoError(t, err)
	err = os.WriteFile(keyFile, keyPEM, 0o600)
	require.NoError(t, err)

	config := &TLSConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   "/nonexistent/ca.pem",
	}

	_, err = config.Build()
	assert.Error(t, err)
}

func TestTLSConfigBuildInvalidCAPEM(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")
	caFile := filepath.Join(tmpDir, "ca.pem")

	certPEM, keyPEM, err := generateTestCertificate()
	require.NoError(t, err)

	err = os.WriteFile(certFile, certPEM, 0o600)
	require.NoError(t, err)
	err = os.WriteFile(keyFile, keyPEM, 0o600)
	require.NoError(t, err)
	err = os.WriteFile(caFile, []byte("invalid ca data"), 0o600)
	require.NoError(t, err)

	config := &TLSConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
	}

	_, err = config.Build()
	assert.Error(t, err)
}

func TestMutualTLSConfigBuildRequireClientCert(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")

	certPEM, keyPEM, err := generateTestCertificate()
	require.NoError(t, err)

	err = os.WriteFile(certFile, certPEM, 0o600)
	require.NoError(t, err)
	err = os.WriteFile(keyFile, keyPEM, 0o600)
	require.NoError(t, err)

	mtc := &MutualTLSConfig{
		TLSConfig: TLSConfig{
			CertFile: certFile,
			KeyFile:  keyFile,
		},
		RequireClientCert: true,
		VerifyClientCert:  false,
	}

	tlsConfig, err := mtc.Build()
	require.NoError(t, err)
	assert.Equal(t, tls.RequireAnyClientCert, tlsConfig.ClientAuth)
}

func TestMutualTLSConfigBuildNoClientCert(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")

	certPEM, keyPEM, err := generateTestCertificate()
	require.NoError(t, err)

	err = os.WriteFile(certFile, certPEM, 0o600)
	require.NoError(t, err)
	err = os.WriteFile(keyFile, keyPEM, 0o600)
	require.NoError(t, err)

	mtc := &MutualTLSConfig{
		TLSConfig: TLSConfig{
			CertFile: certFile,
			KeyFile:  keyFile,
		},
		RequireClientCert: false,
	}

	tlsConfig, err := mtc.Build()
	require.NoError(t, err)
	assert.Equal(t, tls.VerifyClientCertIfGiven, tlsConfig.ClientAuth)
}

func TestNewTLSVerifierWithValidCA(t *testing.T) {
	tmpDir := t.TempDir()
	caFile := filepath.Join(tmpDir, "ca.pem")

	certPEM, _, err := generateTestCertificate()
	require.NoError(t, err)

	err = os.WriteFile(caFile, certPEM, 0o600)
	require.NoError(t, err)

	verifier, err := NewTLSVerifier(caFile)
	require.NoError(t, err)
	assert.NotNil(t, verifier)
	assert.NotNil(t, verifier.caPool)
}

func TestNewTLSVerifierInvalidPEM(t *testing.T) {
	tmpDir := t.TempDir()
	caFile := filepath.Join(tmpDir, "ca.pem")

	err := os.WriteFile(caFile, []byte("invalid pem data"), 0o600)
	require.NoError(t, err)

	verifier, err := NewTLSVerifier(caFile)
	assert.Error(t, err)
	assert.Nil(t, verifier)
}

func TestTLSVerifierVerifyCertificateInvalidCert(t *testing.T) {
	verifier, err := NewTLSVerifier("")
	require.NoError(t, err)

	err = verifier.VerifyCertificate([][]byte{{0x00, 0x01, 0x02}}, nil)
	assert.Error(t, err)
}
