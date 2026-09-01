// Package security reúne primitivas sem dependências de domínio (hash de
// senha), para que tanto auth quanto users possam usá-las sem criar um
// import cycle entre os dois.
package security

import "golang.org/x/crypto/bcrypt"

func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(hash), err
}

func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
