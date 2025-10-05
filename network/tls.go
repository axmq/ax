package network

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

type TLSConfig struct {
	CertFile           string
	KeyFile            string
	CAFile             string
	ClientAuth         tls.ClientAuthType
	MinVersion         uint16
	MaxVersion         uint16
	CipherSuites       []uint16
	InsecureSkipVerify bool
}

func DefaultTLSConfig() *TLSConfig {
	return &TLSConfig{
		ClientAuth:         tls.NoClientCert,
		MinVersion:         tls.VersionTLS13,
		MaxVersion:         tls.VersionTLS13,
		InsecureSkipVerify: false,
	}
}

func (tc *TLSConfig) Build() (*tls.Config, error) {
	if tc.CertFile == "" || tc.KeyFile == "" {
		return nil, ErrInvalidTLSConfig
	}

	cert, err := tls.LoadX509KeyPair(tc.CertFile, tc.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	config := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		ClientAuth:         tc.ClientAuth,
		MinVersion:         tc.MinVersion,
		MaxVersion:         tc.MaxVersion,
		CipherSuites:       tc.CipherSuites,
		InsecureSkipVerify: tc.InsecureSkipVerify,
	}

	if tc.CAFile != "" {
		caCert, err := os.ReadFile(tc.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		config.ClientCAs = caCertPool
		if tc.ClientAuth == tls.NoClientCert {
			config.ClientAuth = tls.RequireAndVerifyClientCert
		}
	}

	return config, nil
}

type TLSVerifier struct {
	caPool         *x509.CertPool
	verifyPeerCert func([][]byte, [][]*x509.Certificate) error
}

func NewTLSVerifier(caFile string) (*TLSVerifier, error) {
	if caFile == "" {
		return &TLSVerifier{}, nil
	}

	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return &TLSVerifier{
		caPool: caPool,
	}, nil
}

func (tv *TLSVerifier) VerifyCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	if tv.verifyPeerCert != nil {
		return tv.verifyPeerCert(rawCerts, verifiedChains)
	}

	if len(rawCerts) == 0 {
		return ErrCertificateVerification
	}

	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	opts := x509.VerifyOptions{
		Roots:         tv.caPool,
		Intermediates: x509.NewCertPool(),
	}

	for _, rawCert := range rawCerts[1:] {
		cert, err := x509.ParseCertificate(rawCert)
		if err != nil {
			continue
		}
		opts.Intermediates.AddCert(cert)
	}

	if _, err := cert.Verify(opts); err != nil {
		return ErrCertificateVerification
	}

	return nil
}

func (tv *TLSVerifier) SetCustomVerifier(fn func([][]byte, [][]*x509.Certificate) error) {
	tv.verifyPeerCert = fn
}

type MutualTLSConfig struct {
	TLSConfig
	RequireClientCert bool
	VerifyClientCert  bool
}

func (mtc *MutualTLSConfig) Build() (*tls.Config, error) {
	config, err := mtc.TLSConfig.Build()
	if err != nil {
		return nil, err
	}

	if mtc.RequireClientCert {
		if mtc.VerifyClientCert {
			config.ClientAuth = tls.RequireAndVerifyClientCert
		} else {
			config.ClientAuth = tls.RequireAnyClientCert
		}
	} else {
		config.ClientAuth = tls.VerifyClientCertIfGiven
	}

	return config, nil
}

func GetPeerCertificates(conn *Connection) ([]*x509.Certificate, error) {
	if !conn.IsTLS() {
		return nil, nil
	}

	state, ok := conn.TLSConnectionState()
	if !ok {
		return nil, nil
	}

	return state.PeerCertificates, nil
}

func GetPeerCommonName(conn *Connection) (string, error) {
	certs, err := GetPeerCertificates(conn)
	if err != nil {
		return "", err
	}

	if len(certs) == 0 {
		return "", nil
	}

	return certs[0].Subject.CommonName, nil
}

func VerifyPeerCertificate(conn *Connection, expectedCN string) error {
	if !conn.IsTLS() {
		return nil
	}

	cn, err := GetPeerCommonName(conn)
	if err != nil {
		return err
	}

	if cn != expectedCN {
		return ErrCertificateVerification
	}

	return nil
}
