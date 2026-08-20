package hasher

import "github.com/alexedwards/argon2id"

func Hash(plain string) (string, error) {
	return argon2id.CreateHash(plain, argon2id.DefaultParams)
}

func Compare(plain, encodedHash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(plain, encodedHash)
}
