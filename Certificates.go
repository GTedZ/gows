package gows

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"time"
)

type certificateHandler struct{}

var Certificates certificateHandler

// GenerateSelfSignedCert creates a TLS certificate and private key in-memory
func (*certificateHandler) GenerateSelfSignedCert(commonName string) (tls.Certificate, []byte, []byte, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour), // 1 year

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		return tls.Certificate{}, nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privKey)})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	return cert, certPEM, keyPEM, err
}

// SaveCertToFiles saves a tls.Certificate to disk as PEM files
func (*certificateHandler) SaveCertToFiles(certPEM, keyPEM []byte, certPath, keyPath string) error {
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return err
	}
	return nil
}

// LoadCertFromFiles loads a tls.Certificate from PEM files
func (*certificateHandler) LoadCertFromFiles(certPath, keyPath string) (tls.Certificate, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.X509KeyPair(certPEM, keyPEM)
}
