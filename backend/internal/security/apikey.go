package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

// EncryptAPIKey cifra a API Key da OpenAI de um professor com AES-256-GCM
// (seção 17) — criptografia autenticada, com nonce aleatório por chamada.
// masterKey deve ter exatamente 32 bytes e nunca é guardada no Postgres:
// vem de fora (variável de ambiente), então um dump do banco sozinho não
// expõe nenhuma chave.
func EncryptAPIKey(plaintext string, masterKey []byte) (ciphertext, nonce []byte, err error) {
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	ciphertext = gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return ciphertext, nonce, nil
}

// DecryptAPIKey reverte EncryptAPIKey. Usado só no momento de chamar a
// OpenAI — o professor nunca recebe a chave descriptografada de volta.
func DecryptAPIKey(ciphertext, nonce, masterKey []byte) (string, error) {
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("could not decrypt API key: invalid ciphertext or key")
	}
	return string(plaintext), nil
}
