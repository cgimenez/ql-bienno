package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"golang.org/x/crypto/ssh"
)

func generateRSAKeys(key_pub *[]byte, key_private *[]byte, bitSize int) error {
	key, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return err
	}

	*key_private = pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	)

	pubKey, err := ssh.NewPublicKey(&key.PublicKey)
	if err != nil {
		return err
	}

	*key_pub = ssh.MarshalAuthorizedKey(pubKey)

	return nil
}
