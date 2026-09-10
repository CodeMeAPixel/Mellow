package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"strconv"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const (
	ivLength        = 16
	tagLength       = 16
	maxDecryptDepth = 5

	noContent     = "[No content]"
	emptyContent  = "[Empty content]"
	undecryptable = "[This content could not be decrypted. Please contact support.]"
)

type Service struct {
	enabled bool
	keys    [][]byte
}

func New(baseKey string, salts []string) *Service {
	if baseKey == "" {
		return &Service{}
	}
	if len(salts) == 0 {
		salts = []string{"mellow-encryption-salt"}
	}
	keys := make([][]byte, 0, len(salts))
	for _, salt := range salts {
		keys = append(keys, pbkdf2.Key([]byte(baseKey), []byte(salt), 10000, 32, sha512.New))
	}
	return &Service{enabled: true, keys: keys}
}

func (s *Service) Enabled() bool { return s != nil && s.enabled }

func (s *Service) Encrypt(text string) string {
	if !s.Enabled() {
		return text
	}
	if strings.TrimSpace(text) == "" {
		return emptyContent
	}
	if s.IsEncrypted(text) {
		return text
	}

	iv := make([]byte, ivLength)
	if _, err := rand.Read(iv); err != nil {
		return text
	}
	block, err := aes.NewCipher(s.keys[0])
	if err != nil {
		return text
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, ivLength)
	if err != nil {
		return text
	}
	sealed := gcm.Seal(nil, iv, []byte(text), nil)
	ct := sealed[:len(sealed)-tagLength]
	tag := sealed[len(sealed)-tagLength:]

	return strings.Join([]string{
		b64(iv),
		strconv.Itoa(tagLength),
		b64(tag),
		b64(ct),
	}, ":")
}

func (s *Service) EncryptPtr(text *string) string {
	if text == nil {
		if !s.Enabled() {
			return noContent
		}
		return noContent
	}
	return s.Encrypt(*text)
}

func (s *Service) Decrypt(payload string) string {
	if !s.Enabled() {
		return payload
	}
	return s.decryptLayered(payload, 0)
}

func (s *Service) decryptLayered(payload string, depth int) string {
	if depth > maxDecryptDepth {
		return undecryptable
	}
	if !s.IsEncrypted(payload) {
		return payload
	}
	for _, key := range s.keys {
		plain, ok := tryDecryptOnce(payload, key)
		if !ok {
			continue
		}
		if s.IsEncrypted(plain) {
			return s.decryptLayered(plain, depth+1)
		}
		return plain
	}
	return undecryptable
}

func (s *Service) IsEncrypted(text string) bool {
	parts := strings.Split(text, ":")
	switch len(parts) {
	case 4:
		if _, err := strconv.Atoi(parts[1]); err != nil {
			return false
		}
		return isB64(parts[0]) && isB64(parts[2]) && isB64(parts[3])
	case 3:
		return isB64(parts[0]) && isB64(parts[1]) && isB64(parts[2])
	default:
		return false
	}
}

func tryDecryptOnce(payload string, key []byte) (string, bool) {
	parts := strings.Split(payload, ":")

	var ivB64, tagB64, ctB64 string
	tagLen := tagLength
	switch len(parts) {
	case 4:
		n, err := strconv.Atoi(parts[1])
		if err != nil {
			return "", false
		}
		tagLen = n
		ivB64, tagB64, ctB64 = parts[0], parts[2], parts[3]
	case 3:
		ivB64, tagB64, ctB64 = parts[0], parts[1], parts[2]
	default:
		return "", false
	}

	iv, err := base64.StdEncoding.DecodeString(ivB64)
	if err != nil || len(iv) != ivLength {
		return "", false
	}
	tag, err := base64.StdEncoding.DecodeString(tagB64)
	if err != nil {
		return "", false
	}
	ct, err := base64.StdEncoding.DecodeString(ctB64)
	if err != nil {
		return "", false
	}
	if len(parts) == 3 {
		tagLen = len(tag)
	}

	if tagLen != tagLength || len(tag) != tagLength {
		return "", false
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", false
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, ivLength)
	if err != nil {
		return "", false
	}
	plain, err := gcm.Open(nil, iv, append(ct, tag...), nil)
	if err != nil {
		return "", false
	}
	return string(plain), true
}

func b64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

func isB64(s string) bool {
	if s == "" {
		return false
	}
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}
