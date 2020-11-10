package authtoken

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	testKey      = "super-secret-key"
	testIssuer   = "unit-tester"
	testAudience = "this-unit-test"
)

func TestToken(t *testing.T) {
	token, err := SignToken(testKey, testIssuer, testAudience, map[string]interface{}{
		"iam": "me",
	}, map[string]interface{}{
		"pw": "super-secret-password",
		"pi": 3.14,
		"hg": 42,
		"bs": []byte("some-bytes"),
	})

	require.NoError(t, err)

	claims, err := ValidateToken(testKey, testIssuer, testAudience, token)
	require.NoError(t, err)
	require.Equal(t, "me", claims["iam"].(string))
	require.Equal(t, testIssuer, claims["iss"].(string))
	require.Equal(t, "super-secret-password", claims["pw"].(string))
	require.Equal(t, "3.14e+00", claims["pi"].(string))
	require.Equal(t, "42", claims["hg"].(string))
	require.Equal(t, "some-bytes", claims["bs"].(string))
}

func TestPlainClaimWithEncPrefix(t *testing.T) {
	token, err := SignToken(testKey, testIssuer, testAudience, map[string]interface{}{
		encryptedClaimPrefix + "iam": "should-fail",
	}, map[string]interface{}{
		"pw": "super-secret-password",
	})

	require.Error(t, err)
	require.Empty(t, token)
}

func TestUnsupportedType(t *testing.T) {
	token, err := SignToken(testKey, testIssuer, testAudience, map[string]interface{}{
		"iam": "me",
	}, map[string]interface{}{
		"ohno": uint8(4),
	})

	require.Error(t, err)
	require.Empty(t, token)
}

func TestInvalidIssuer(t *testing.T) {
	token, err := SignToken(testKey, testIssuer, testAudience, map[string]interface{}{
		"iam": "me",
	}, map[string]interface{}{
		"pw": "super-secret-password",
		"pi": 3.14,
		"hg": 42,
		"bs": []byte("some-bytes"),
	})

	require.NoError(t, err)

	claims, err := ValidateToken(testKey, "wrong-issuer", testAudience, token)
	require.Error(t, err)
	require.Empty(t, claims)
}

func TestInvalidAudience(t *testing.T) {
	token, err := SignToken(testKey, testIssuer, testAudience, map[string]interface{}{
		"iam": "me",
	}, map[string]interface{}{
		"pw": "super-secret-password",
		"pi": 3.14,
		"hg": 42,
		"bs": []byte("some-bytes"),
	})

	require.NoError(t, err)

	claims, err := ValidateToken(testKey, testIssuer, "wrong-audience", token)
	require.Error(t, err)
	require.Empty(t, claims)
}

func TestInvalidNBF(t *testing.T) {
	token, err := SignToken(testKey, testIssuer, testAudience, map[string]interface{}{
		"iam": "me",
		"nbf": time.Now().Unix() + 1000,
	}, map[string]interface{}{
		"pw": "super-secret-password",
		"pi": 3.14,
		"hg": 42,
		"bs": []byte("some-bytes"),
	})

	require.NoError(t, err)

	claims, err := ValidateToken(testKey, testIssuer, testAudience, token)
	require.Error(t, err)
	require.Empty(t, claims)
}
