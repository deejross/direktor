package server

import (
	"fmt"

	"github.com/deejross/direktor/internal/config"
	"github.com/deejross/direktor/pkg/authtoken"
	"github.com/deejross/direktor/pkg/ldapcli"
	"github.com/gin-gonic/gin"
	"github.com/go-ldap/ldap/v3"
)

// AuthTokenRequest object.
type AuthTokenRequest struct {
	Address         string `json:"address"`
	BaseDN          string `json:"baseDN"`
	Username        string `json:"username,omitempty"`
	Password        string `json:"password,omitempty"`
	StartTLS        *bool  `json:"startTLS,omitempty"`
	SkipVerify      *bool  `json:"skipVerify,omitempty"`
	PageSize        *int   `json:"pageSize,omitempty"`
	FollowReferrals *bool  `json:"followReferrals,omitempty"`
}

// Validate the request.
func (r *AuthTokenRequest) Validate() error {
	if len(r.Address) == 0 {
		return fmt.Errorf("address is a required field")
	}
	if len(r.BaseDN) == 0 {
		return fmt.Errorf("baseDN is a required field")
	}
	return nil
}

// AuthTokenResponse object.
type AuthTokenResponse struct {
	Token string `json:"token"`
}

func handleAuthToken(c *gin.Context) {
	req := &AuthTokenRequest{}
	if err := c.ShouldBind(req); err != nil {
		newError(c, 400, err)
		return
	}

	if err := req.Validate(); err != nil {
		newError(c, 400, err)
		return
	}

	plainClaims := map[string]interface{}{
		claimBaseDN:       req.BaseDN,
		claimBindUsername: req.Username,
	}

	ldapConf := ldapcli.NewConfig(req.Address, req.BaseDN)
	ldapConf.BindUsername = req.Username
	ldapConf.BindPassword = req.Password

	if req.FollowReferrals != nil {
		plainClaims[claimFollowReferrals] = *req.FollowReferrals
		ldapConf.FollowReferrals = *req.FollowReferrals
	}
	if req.PageSize != nil && *req.PageSize > 0 {
		plainClaims[claimPageSize] = *req.PageSize
		ldapConf.PageSize = *req.PageSize
	}
	if req.SkipVerify != nil {
		plainClaims[claimSkipVerify] = *req.SkipVerify
		ldapConf.SkipVerify = *req.SkipVerify
	}
	if req.StartTLS != nil {
		plainClaims[claimStartTLS] = *req.StartTLS
		ldapConf.StartTLS = *req.StartTLS
	}

	cli, err := ldapcli.Dial(ldapConf)
	if err != nil {
		if e, ok := err.(*ldap.Error); ok {
			if e.ResultCode == ldap.LDAPResultInvalidCredentials {
				newError(c, 401, e)
				return
			}

			newError(c, 400, e)
			return
		}

		newError(c, 400, err)
		return
	}
	cli.Close()

	encryptedClaims := map[string]interface{}{
		claimBindPassword: req.Password,
	}

	conf, err := config.Get()
	if err != nil {
		newError(c, 500, err)
		return
	}

	token, err := authtoken.SignToken(conf.SecretKey, tokenIssuer, req.Address, plainClaims, encryptedClaims)
	if err != nil {
		newError(c, 500, fmt.Errorf("could not sign token: %v", err))
		return
	}

	c.JSON(200, &AuthTokenResponse{
		Token: token,
	})
}

func handleAuthTokenCheck(c *gin.Context) {
	cli := ldapClient(c)
	if cli != nil {
		cli.Close()
		c.JSON(200, gin.H{
			"result": "OK",
		})
	}
}
