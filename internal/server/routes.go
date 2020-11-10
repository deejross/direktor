package server

import (
	"fmt"
	"strings"

	"github.com/deejross/direktor/internal/config"
	"github.com/deejross/direktor/pkg/authtoken"
	"github.com/deejross/direktor/pkg/ldapcli"
	"github.com/gin-gonic/gin"
	"github.com/go-ldap/ldap/v3"
	"go.uber.org/zap"
)

const (
	claimBindUsername    = "bun"
	claimBindPassword    = "bpw"
	claimStartTLS        = "stls"
	claimSkipVerify      = "skvy"
	claimBaseDN          = "bdn"
	claimPageSize        = "psz"
	claimFollowReferrals = "fref"
)

func registerRoutes(router *gin.Engine) {
	v1 := router.Group("/v1")

	// auth endpoints
	v1.GET("/auth/token", handleAuthTokenCheck)
	v1.POST("/auth/token", handleAuthToken)
}

func newError(c *gin.Context, code int, err error) {
	c.AbortWithStatusJSON(code, gin.H{
		"code":  code,
		"error": err.Error(),
	})
}

// ldapClient retrieves the requested LDAP client via the Authorization header.
// Any errors encountered will be sent back as a JSON response and this function will return nil.
func ldapClient(c *gin.Context) *ldapcli.Client {
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) == 0 {
		newError(c, 401, fmt.Errorf("Authorization header required"))
		return nil
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		newError(c, 400, fmt.Errorf("unknown Authorization method"))
		return nil
	}

	ldapAddress := c.GetHeader("X-Ldap-Address")
	if len(ldapAddress) == 0 {
		newError(c, 400, fmt.Errorf("X-Ldap-Address header required"))
		return nil
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	conf, err := config.Get()
	if err != nil {
		log.Error("could not get config", zap.Error(err))
		newError(c, 500, fmt.Errorf("configuration error, please see server logs for more information"))
		return nil
	}

	claims, err := authtoken.ValidateToken(conf.SecretKey, tokenIssuer, ldapAddress, token)
	if err != nil {
		newError(c, 401, err)
		return nil
	}

	bindBaseDN, ok := claims[claimBaseDN]
	if !ok {
		newError(c, 400, fmt.Errorf("token does not contain `%s` claim", claimBaseDN))
	}

	ldapConf := ldapcli.NewConfig(ldapAddress, bindBaseDN.(string))

	if val, ok := claims[claimBindUsername]; ok {
		ldapConf.BindUsername = val.(string)
	}
	if val, ok := claims[claimBindPassword]; ok {
		ldapConf.BindPassword = val.(string)
	}
	if val, ok := claims[claimFollowReferrals]; ok {
		ldapConf.FollowReferrals = val.(bool)
	}
	if val, ok := claims[claimStartTLS]; ok {
		ldapConf.StartTLS = val.(bool)
	}
	if val, ok := claims[claimSkipVerify]; ok {
		ldapConf.SkipVerify = val.(bool)
	}
	if val, ok := claims[claimPageSize]; ok {
		ldapConf.PageSize = val.(int)
	}

	cli, err := ldapcli.Dial(ldapConf)
	if err != nil {
		if e, ok := err.(*ldap.Error); ok {
			if e.ResultCode == ldap.LDAPResultInvalidCredentials {
				newError(c, 401, e)
				return nil
			}

			newError(c, 400, e)
			return nil
		}

		newError(c, 400, err)
		return nil
	}

	return cli
}
