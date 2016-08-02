package server

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/getlantern/go-update"
)

const (
	privateKeyEnv = `PRIVATE_KEY`
)

var (
	privateKeyFile  string
	rsaPrivateKey   *rsa.PrivateKey
	rsaPrivateKeyMu sync.Mutex
)

func init() {
	privateKeyFile = os.Getenv(privateKeyEnv)
}

func SetPrivateKey(s string) {
	privateKeyFile = s
	if _, err := privateKey(); err != nil {
		log.Fatal(err)
	}
}

func checksumForFile(file string) (checksumHex string, err error) {
	var checksum []byte
	if checksum, err = update.ChecksumForFile(file); err != nil {
		return "", err
	}
	checksumHex = hex.EncodeToString(checksum)
	return checksumHex, nil
}

func privateKey() (*rsa.PrivateKey, error) {
	var err error

	rsaPrivateKeyMu.Lock()
	defer rsaPrivateKeyMu.Unlock()

	if rsaPrivateKey != nil {
		return rsaPrivateKey, nil
	}

	if privateKeyFile == "" {
		log.Fatalf("Missing %s environment variable.", privateKeyEnv)
	}

	// Loading private key
	var pb []byte
	var fpk *os.File

	if fpk, err = os.Open(privateKeyFile); err != nil {
		return nil, fmt.Errorf("Could not open private key: %q", err)
	}
	defer fpk.Close()

	if pb, err = ioutil.ReadAll(fpk); err != nil {
		return nil, fmt.Errorf("Could not read private key: %q", err)
	}

	// Decoding PEM key.
	pemBlock, _ := pem.Decode(pb)

	rsaPrivateKey, err = x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
	if err != nil {
		return nil, err
	}

	return rsaPrivateKey, nil
}

// Sign creates a signatures for a byte array.
func Sign(hashedMessage []byte) ([]byte, error) {
	pk, err := privateKey()
	if err != nil {
		return nil, err
	}

	// Checking message signature.
	var signature []byte
	if signature, err = rsa.SignPKCS1v15(rand.Reader, pk, crypto.SHA256, hashedMessage); err != nil {
		return nil, err
	}

	return signature, nil
}

func signatureForFile(file string) (string, error) {
	checksum, err := checksumForFile(file)
	if err != nil {
		return "", err
	}

	checksumHex, err := hex.DecodeString(checksum)
	if err != nil {
		return "", err
	}

	signature, err := Sign(checksumHex)
	if err != nil {
		return "", fmt.Errorf("Could not sign file %q: %q", file, err)
	}

	return hex.EncodeToString(signature), nil
}
