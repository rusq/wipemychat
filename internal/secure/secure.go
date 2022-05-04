// Package secure provides encryption and decryption functions.
//
// Encryption implementation
//
// "Salt" is a fixed 256 byte array of pseudo-random values, taken from
// /dev/urandom.
//
// Encryption key is a 256-bit value (32 bytes).
//
// Encryption key is derived in the following manner:
//
//   1. Repeat bytes of the passphrase to form 32 bytes of the Key
//   2. Take the first byte of the passphrase and use it for the value of Offset in
//      the Salt array.
//   3. For each byte of the key, and `i` being the counter:
//       - `Key[i] ^= Salt[(Offset+i)%Key_length]
//
// Then the plain text is encrypted with the Key using AES-256 in GCM and signed
// together with additional data.
//
// Then additional data, nonce and ciphertext are packed into the following
// sequence of bytes:
//
//   |_|__...__|_________|__...__|
//    ^    ^        ^        ^
//    |    |        |        +- ciphertext, n bytes.
//    |    |        +---------- nonce, (nonceSz bytes)
//    |    +------------------- additinal data, m bytes, (maxDataSz bytes),
//    +------------------------ additional data length value (adlSz bytes).
//
// After this, packed byte sequence is armoured with base64 and the signature
// prefix added to it to distinct it from the plain text.
package secure

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	nonceSz   = 12               // bytes, nonce sz
	keyBits   = 256              // encryption gKey size.
	keySz     = keyBits / 8      // bytes, gKey size
	adlSz     = 1                // bytes, size of additional data length field
	maxDataSz = 1<<(adlSz*8) - 1 // bytes, max additional data size (this is maximum that can fit into (adlSz) bytes)

	signature = "TGD." // used to identified encrypted strings
	sigSz     = len(signature)
)

// salt will be used to XOR the gKey which we generate by padding the passphrase.
var salt = [keySz * 8]byte{
	0x1a, 0x98, 0x15, 0x70, 0xbf, 0x57, 0x16, 0x35, 0xba, 0x78, 0x1e, 0xbc,
	0x97, 0x09, 0x24, 0x47, 0xe7, 0xa6, 0xac, 0x72, 0x0d, 0x60, 0x28, 0x8b,
	0x40, 0x13, 0x02, 0x0d, 0xd6, 0x38, 0xa3, 0xfa, 0x95, 0x14, 0xc6, 0x7d,
	0x65, 0x3d, 0xb2, 0xd9, 0x86, 0x4f, 0x61, 0x5f, 0xa5, 0xe7, 0xdc, 0x30,
	0x52, 0x49, 0x0c, 0x6d, 0x1a, 0xea, 0x2b, 0x5b, 0xf6, 0x4a, 0x5f, 0xd2,
	0xfd, 0x01, 0x1a, 0xc8, 0x48, 0x68, 0xcf, 0x7b, 0xfa, 0x64, 0xc7, 0x46,
	0x82, 0xdc, 0x78, 0xb6, 0xc0, 0x80, 0x07, 0xb5, 0xa0, 0x79, 0x3f, 0xcb,
	0xe5, 0xee, 0x55, 0x72, 0x74, 0x66, 0x6d, 0xe4, 0x8e, 0xed, 0xd1, 0xff,
	0xba, 0x6b, 0x51, 0xf7, 0xca, 0xfe, 0x43, 0x3f, 0xbd, 0x37, 0xb5, 0x37,
	0xa3, 0xa4, 0x05, 0x44, 0xd4, 0x1f, 0xb9, 0xd9, 0xc0, 0x2f, 0x41, 0xa6,
	0xe9, 0x14, 0x6b, 0xef, 0xdd, 0x67, 0x0d, 0x5e, 0x10, 0x31, 0xca, 0xdc,
	0xd1, 0x42, 0xdd, 0x9d, 0xef, 0x14, 0x7f, 0xff, 0x4d, 0x03, 0x65, 0xdc,
	0x66, 0x5d, 0x92, 0x4c, 0x23, 0x89, 0xf7, 0x62, 0x9d, 0x2a, 0x06, 0xe1,
	0x66, 0x0a, 0x47, 0x24, 0xd3, 0x08, 0xc1, 0x04, 0x45, 0xb5, 0xcd, 0x1c,
	0x61, 0x08, 0x52, 0xf5, 0x4e, 0xb8, 0xbd, 0x47, 0x69, 0x30, 0xec, 0x02,
	0x61, 0xf9, 0xd8, 0xc9, 0x93, 0x20, 0x8b, 0x33, 0xe9, 0x96, 0xab, 0xd4,
	0x43, 0x91, 0x59, 0xe0, 0x4e, 0x45, 0x5c, 0xda, 0x57, 0x0e, 0x12, 0x77,
	0xa4, 0xe2, 0x0d, 0x7e, 0xee, 0xe3, 0x2e, 0x80, 0x98, 0x39, 0xd1, 0x98,
	0x34, 0x4e, 0x3f, 0xff, 0xcf, 0xca, 0x1f, 0xe6, 0x36, 0xfc, 0x58, 0x12,
	0xfd, 0x8e, 0x28, 0x83, 0x74, 0xbc, 0xf9, 0xeb, 0xf8, 0xd3, 0x4f, 0x39,
	0x35, 0x74, 0x5d, 0xa7, 0x65, 0x64, 0x0b, 0x13, 0x38, 0x0e, 0x4b, 0x63,
	0xcf, 0x47, 0x64, 0xf2,
}

var (
	ErrNotEncrypted    = errors.New("string not encrypted")
	ErrNoEncryptionKey = errors.New("no encryption gKey")
	ErrDataOverflow    = errors.New("additional data overflow")
	ErrInvalidKeySz    = errors.New("invalid Key size")
)

// CipherError indicates that there was an error during decrypting of
// ciphertext.
type CipherError struct {
	Err error
}

func (e *CipherError) Error() string {
	return e.Err.Error()
}

func (e *CipherError) Unwrap() error {
	return e.Err
}

func (e *CipherError) Is(target error) bool {
	t, ok := target.(*CipherError)
	if !ok {
		return false
	}
	return e.Err.Error() == t.Err.Error()
}

type CorruptError struct {
	Value []byte
}

func (e *CorruptError) Error() string {
	return "corrupt packed data"
}

func (e *CorruptError) Is(target error) bool {
	t, ok := target.(*CorruptError)
	if !ok {
		return false
	}
	return bytes.Equal(t.Value, e.Value)
}

var gKey []byte

// setGlobalKey sets the encryption gKey globally.
func setGlobalKey(k []byte) error {
	if len(k) != keySz {
		return ErrInvalidKeySz
	}
	gKey = k
	return nil
}

func SetPassphrase(b []byte) error {
	k, err := deriveKey(b)
	if err != nil {
		return err
	}
	return setGlobalKey(k)
}

// deriveKey interpolates the passphrase value to the gKey size and xors it with salt.
func deriveKey(pass []byte) ([]byte, error) {
	if len(pass) == 0 {
		return nil, errors.New("empty passphrase")
	}
	if len(pass) > keySz {
		return nil, errors.New("passphrase is too big")
	}

	var key = make([]byte, keySz)
	var startOffset = int(pass[0]) // starting offset in salt is the first byte of the password

	for i := range key {
		key[i] = pass[i%len(pass)] ^ salt[(i+startOffset)%len(salt)]
	}
	return key, nil
}

// Encrypt encrypts the plain text password to use in the configuration file
// with the gKey generated by KeyFn.
func Encrypt(plaintext string) (string, error) {
	return encrypt(plaintext, gKey, nil)
}

// Decrypt attempts to decrypt the string and return the password.
// In case s is not an encrypted string, ErrNotEncrypted returned along with
// original string.
func Decrypt(s string) (string, error) {
	return decrypt(s, gKey)
}

// EncryptWithPassphrase encrypts plaintext with the provided passphrase
func EncryptWithPassphrase(plaintext string, passphrase []byte) (string, error) {
	key, err := deriveKey(passphrase)
	if err != nil {
		return "", err
	}
	return encrypt(plaintext, key, nil)
}

// DecryptWithPassphrase attempts to descrypt string with the provided MAC
// address.
func DecryptWithPassphrase(s string, passphrase []byte) (string, error) {
	key, err := deriveKey(passphrase)
	if err != nil {
		return "", err
	}
	return decrypt(s, key)
}

// Encrypt encrypts the plain text password to use in the configuration file.
func encrypt(plaintext string, key []byte, additionalData []byte) (string, error) {
	if len(key) == 0 {
		return "", ErrNoEncryptionKey
	}
	if len(key) != keySz {
		return "", ErrInvalidKeySz
	}
	if len(plaintext) == 0 {
		return "", errors.New("nothing to encrypt")
	}
	if len(additionalData) > maxDataSz {
		return "", fmt.Errorf("size of additional data can't exceed %d B", maxDataSz)
	}

	gcm, err := initGCM(key)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, nonceSz)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), additionalData)

	// return signature + base64.StdEncoding.EncodeToString(data), nil
	packed, err := pack(ciphermsg{nonce, ciphertext, additionalData})
	if err != nil {
		return "", err
	}

	return armor(packed), nil
}

// initGCM initialises the Galois/Counter Mode
func initGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func pack(cm ciphermsg) ([]byte, error) {
	if len(cm.nonce) == 0 {
		return nil, errors.New("pack: empty nonce")
	}
	if len(cm.ciphertext) == 0 {
		return nil, errors.New("pack: no ciphertext")
	}
	dataLen := len(cm.additionalData)
	if dataLen > maxDataSz {
		return nil, ErrDataOverflow
	}

	packed := make([]byte, nonceSz+len(cm.ciphertext)+1+dataLen)
	packed[0] = byte(dataLen)
	if dataLen > 0 {
		copy(packed[adlSz:], cm.additionalData)
	}
	copy(packed[adlSz+dataLen:], cm.nonce)
	copy(packed[adlSz+dataLen+nonceSz:], cm.ciphertext)

	return packed, nil
}

func armor(packed []byte) string {
	return signature + base64.StdEncoding.EncodeToString(packed)
}

func unarmor(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if len(s) < sigSz || s[0:sigSz] != signature {
		return nil, ErrNotEncrypted
	}
	packed, err := base64.StdEncoding.DecodeString(s[sigSz:])
	if err != nil {
		return nil, err
	}
	return packed, nil
}

type ciphermsg struct {
	nonce          []byte
	ciphertext     []byte
	additionalData []byte
}

func unpack(packed []byte) (*ciphermsg, error) {
	if len(packed) == 0 {
		return nil, errors.New("unpack: empty input")
	}
	var (
		dataLen   = int(packed[0])
		payloadSz = len(packed) - adlSz - nonceSz // payload is data + ct size
	)
	if dataLen > payloadSz || payloadSz-dataLen == 0 {
		return nil, &CorruptError{packed}
	}
	cm := &ciphermsg{
		nonce:      packed[adlSz+dataLen : adlSz+dataLen+nonceSz],
		ciphertext: packed[adlSz+dataLen+nonceSz:],
	}
	if dataLen > 0 {
		cm.additionalData = packed[adlSz : adlSz+dataLen]
	}
	return cm, nil
}

func decrypt(s string, key []byte) (string, error) {
	packed, err := unarmor(s)
	if err != nil {
		if err == ErrNotEncrypted {
			return s, err
		}
		return "", err // other error
	}
	if len(key) == 0 {
		return "", ErrNoEncryptionKey
	}
	cm, err := unpack(packed)
	if err != nil {
		return "", err
	}
	aesgcm, err := initGCM(key)
	if err != nil {
		return "", err
	}

	plaintext, err := aesgcm.Open(nil, cm.nonce, cm.ciphertext, cm.additionalData)
	if err != nil {
		return "", &CipherError{err}
	}
	return string(plaintext), nil
}

// IsDecryptError returns true if there was a decryption error or corrupt data
// error and false if it's a different kind of error.
func IsDecryptError(err error) bool {
	switch err.(type) {
	case *CipherError, *CorruptError:
		return true
	}
	return false
}
