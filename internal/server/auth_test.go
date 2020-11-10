package server

import (
	"testing"

	"github.com/deejross/direktor/pkg/ldapmockserver"
	"github.com/stretchr/testify/require"
)

func TestAuthToken(t *testing.T) {
	req := AuthTokenRequest{
		Address:  ldapAddress,
		BaseDN:   ldapmockserver.TestBaseDN,
		Username: ldapmockserver.TestBindDN,
		Password: ldapmockserver.TestBindPW,
	}

	resp := &AuthTokenResponse{}
	w, err := newRequest("POST", "/v1/auth/token", "", "", req, resp)

	require.NoError(t, err)
	require.Equal(t, 200, w.StatusCode)
	require.NotEmpty(t, resp.Token)

	t.Run("Check", func(t *testing.T) {
		w, err := newRequest("GET", "/v1/auth/token", resp.Token, ldapAddress, nil, nil)
		require.NoError(t, err)
		require.Equal(t, 200, w.StatusCode)
	})
}
