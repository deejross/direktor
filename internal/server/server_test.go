package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/deejross/direktor/internal/config"
	"github.com/deejross/direktor/pkg/ldapmockserver"
	"github.com/gin-gonic/gin"
)

const (
	ldapAddress   = "ldap://127.0.0.1:10389"
	testSecretKey = "super-secret-test-key"
)

var (
	router *gin.Engine
)

func TestMain(m *testing.M) {
	// start the LDAP server
	stopCh, err := ldapmockserver.Start("127.0.0.1:10389")
	if err != nil {
		log.Fatal(err.Error())
	}

	// setup the server config
	config.Set(&config.Config{
		SecretKey: testSecretKey,
	})

	// setup the API server
	router = setupRouter()

	// run the tests
	code := m.Run()

	// cleanup
	stopCh <- struct{}{}
	os.Exit(code)
}

func newRequest(method, path, token, ldapAddress string, body interface{}, v interface{}) (*http.Response, error) {
	var req *http.Request
	var err error
	var bodyBuf *bytes.Buffer

	if body != nil {
		bodyBuf = &bytes.Buffer{}
		if err := json.NewEncoder(bodyBuf).Encode(body); err != nil {
			return nil, err
		}

		req, err = http.NewRequest(method, path, bodyBuf)
	} else {
		req, err = http.NewRequest(method, path, nil)
	}

	w := httptest.NewRecorder()

	if err != nil {
		return nil, err
	}

	if bodyBuf != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if len(token) > 0 {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if len(ldapAddress) > 0 {
		req.Header.Set("X-Ldap-Address", ldapAddress)
	}

	router.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		if w.Body.Len() > 0 && v != nil {
			if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
				return resp, err
			}
		}
	} else if w.Body.Len() > 0 {
		m := map[string]interface{}{}
		if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
			return resp, err
		}
		if errStr, ok := m["error"]; ok {
			return resp, fmt.Errorf(errStr.(string))
		}
		return resp, fmt.Errorf("unknown error")
	}

	return resp, nil
}
