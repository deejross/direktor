package authtoken

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
)

const (
	encryptedClaimPrefix = "enc-"
)

var (
	signingMethod = jwt.SigningMethodHS256
)

// SignToken signs a new JWT token with the given claims, optionally with encrypted claims.
func SignToken(keyStr, issuer, audience string, plainClaims, encryptedClaims map[string]interface{}) (string, error) {
	key := hashKey(keyStr)

	claims := jwt.MapClaims{
		"nbf": time.Now().Unix(),
		"iss": issuer,
		"aud": audience,
	}

	if plainClaims != nil {
		for k, v := range plainClaims {
			if strings.HasPrefix(k, encryptedClaimPrefix) {
				return "", fmt.Errorf("claim: %s: cannot have '%s' prefix: this is reserved for encrypted claims", k, encryptedClaimPrefix)
			}

			claims[k] = v
		}
	}

	if encryptedClaims != nil {
		for k, iv := range encryptedClaims {
			var encStr string
			var err error

			switch v := iv.(type) {
			case string:
				encStr, err = encrypt(key, []byte(v))
			case []byte:
				encStr, err = encrypt(key, v)
			case int:
				encStr, err = encrypt(key, []byte(strconv.Itoa(v)))
			case float64:
				encStr, err = encrypt(key, []byte(strconv.FormatFloat(v, 'e', -1, 64)))
			default:
				return "", fmt.Errorf("unable to encrypt claim: %s: unsupported type", k)
			}

			if err != nil {
				return "", fmt.Errorf("unable to encrypt claim: %s: %v", k, err)
			}

			claims[encryptedClaimPrefix+k] = encStr
		}
	}

	token := jwt.NewWithClaims(signingMethod, claims)
	return token.SignedString(key)
}

// ValidateToken validates the given JWT and returns its claims.
func ValidateToken(keyStr, issuer, audience, token string) (map[string]interface{}, error) {
	key := hashKey(keyStr)

	tokenObj, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		return key, nil
	})

	if err != nil {
		return nil, err
	}

	tokenClaims, ok := tokenObj.Claims.(jwt.MapClaims)
	if !ok || !tokenObj.Valid {
		return nil, fmt.Errorf("token validation failed")
	}

	if tokenClaims["iss"].(string) != issuer {
		return nil, fmt.Errorf("token issuer invalid")
	}

	if int64(tokenClaims["nbf"].(float64)) > time.Now().Unix() {
		return nil, fmt.Errorf("token not yet valid")
	}

	if tokenClaims["aud"].(string) != audience {
		return nil, fmt.Errorf("token invalid audience")
	}

	claims := map[string]interface{}{}

	for k, v := range tokenClaims {
		if strings.HasPrefix(k, encryptedClaimPrefix) {
			bs, err := decrypt(key, v.(string))
			if err != nil {
				return nil, fmt.Errorf("unable to decrypt claim: %s: %v", k, err)
			}

			claims[strings.TrimPrefix(k, encryptedClaimPrefix)] = string(bs)
		} else {
			claims[k] = v
		}
	}

	return claims, nil
}

func initGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm, nil
}

func hashKey(key string) []byte {
	h := sha512.New()
	h.Write([]byte(key))
	sum := h.Sum(nil)
	return sum[:32]
}

func randBytes(length int) []byte {
	b := make([]byte, length)
	rand.Read(b)
	return b
}

func encrypt(key []byte, plaintext []byte) (string, error) {
	gcm, err := initGCM(key)
	if err != nil {
		return "", err
	}

	nonce := randBytes(gcm.NonceSize())
	bs := gcm.Seal(nil, nonce, plaintext, nil)
	bs = append(nonce, bs...)
	return base64.RawStdEncoding.EncodeToString(bs), nil
}

func decrypt(key []byte, ciphertext string) ([]byte, error) {
	gcm, err := initGCM(key)
	if err != nil {
		return nil, err
	}

	ciphertextBS, err := base64.RawStdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("could not decode cipher: %v", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertextBS) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertextBS[0:nonceSize]
	msg := ciphertextBS[nonceSize:]
	bs, err := gcm.Open(nil, nonce, msg, nil)
	if err != nil {
		return nil, err
	}

	return bs, nil
}
