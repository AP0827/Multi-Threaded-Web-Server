package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "certgen error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("certgen", flag.ContinueOnError)
	certOut := fs.String("cert-out", "certs/mtws-local.crt", "certificate output path")
	keyOut := fs.String("key-out", "certs/mtws-local.key", "private key output path")
	hosts := fs.String("hosts", "localhost,127.0.0.1,::1", "comma-separated DNS names or IP addresses")
	validFor := fs.Duration("valid-for", 365*24*time.Hour, "certificate validity duration")
	force := fs.Bool("force", false, "overwrite existing certificate files")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *validFor <= 0 {
		return fmt.Errorf("valid-for must be positive")
	}
	if !*force {
		if fileExists(*certOut) || fileExists(*keyOut) {
			return fmt.Errorf("output file already exists; rerun with -force to overwrite")
		}
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generate private key: %w", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("generate serial number: %w", err)
	}

	now := time.Now()
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"MTWS Local Development"},
			CommonName:   "MTWS Local Development Certificate",
		},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(*validFor),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	for _, host := range splitHosts(*hosts) {
		if ip := net.ParseIP(host); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
			continue
		}
		template.DNSNames = append(template.DNSNames, host)
	}
	if len(template.DNSNames) == 0 && len(template.IPAddresses) == 0 {
		return fmt.Errorf("at least one DNS name or IP address is required")
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("create certificate: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(*certOut), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*keyOut), 0o755); err != nil {
		return err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	if err := os.WriteFile(*certOut, certPEM, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(*keyOut, keyPEM, 0o600); err != nil {
		return err
	}

	fmt.Printf("Wrote certificate: %s\n", *certOut)
	fmt.Printf("Wrote private key: %s\n", *keyOut)
	fmt.Println("Use MTWS_TLS_CERT_FILE and MTWS_TLS_KEY_FILE to enable TLS.")
	return nil
}

func splitHosts(value string) []string {
	parts := strings.Split(value, ",")
	hosts := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			hosts = append(hosts, part)
		}
	}
	return hosts
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
