package ldapcli

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"golang.org/x/text/encoding/unicode"
)

const (
	// AttributeCommonName is the name for the CN (common name) attribute.
	AttributeCommonName = "cn"

	// AttributeDepartment is the name for the department attribuite.
	AttributeDepartment = "department"

	// AttributeDescription is the name for the description attribute.
	AttributeDescription = "description"

	// AttributeDisplayName is name for the display name attribute.
	AttributeDisplayName = "displayName"

	// AttributeDistinguishedName is the name for the DN (distinguished name) attribute.
	AttributeDistinguishedName = "distinguishedName"

	// AttributeMail is the name for the mail attribute.
	AttributeMail = "mail"

	// AttributeMemberOf is the name of the memberOf attribute.
	AttributeMemberOf = "memberOf"

	// AttributeMemberOfNested is the name of the memberOf attribute that includes nested group members.
	AttributeMemberOfNested = "memberOf:1.2.840.113556.1.4.1941:"

	// AttributeObjectClass is the name for the object class attribute.
	AttributeObjectClass = "objectClass"

	// AttributeSAMAccountName is the name for the Active Directory sAMAccountName attribute.
	AttributeSAMAccountName = "sAMAccountName"

	// AttributeUnicodePassword is the name of the double-quoted UTF16 encoded password attribute.
	AttributeUnicodePassword = "unicodePwd"

	// AttributeUserPrincipalName is the name of the userPrincipalName attribute.
	AttributeUserPrincipalName = "userPrincipalName"

	// ObjectClassGroup is the name of the group object class.
	ObjectClassGroup = "group"

	// ObjectClassPerson is the name of the person object class.
	ObjectClassPerson = "person"
)

// Config object for client.
type Config struct {
	Address           string // address with ldap:// or ldaps:// protocol prefix
	BindUsername      string // optional, bind as username or userPrincipalName
	BindPassword      string // optional, bind with password
	StartTLS          bool   // should the connection attempt to STARTTLS
	SkipVerify        bool   // ignore insecure TLS validation errors
	BaseDN            string // base DN for searching
	PageSize          int    // the number of results to request per page, default: 1000
	DefaultTimeLimit  int    // default time limit to wait for results, default: 0 (no time limit)
	FollowReferrals   bool   // should searches that return referrals be followed, default: true
	userPrincipalName string // calculated userPrincipalName for binding
}

// NewConfig returns a new Config object with defaults set.
func NewConfig(address, baseDN string) *Config {
	return &Config{
		Address:         address,
		BaseDN:          baseDN,
		PageSize:        1000,
		FollowReferrals: true,
	}
}

// Validate the Config has all the required fields.
func (c *Config) Validate() error {
	if len(c.Address) == 0 {
		return fmt.Errorf("Address is a required field")
	}

	if len(c.BaseDN) == 0 {
		return fmt.Errorf("BaseDN is a required field")
	}

	if c.PageSize < 100 {
		c.PageSize = 100
	} else if c.PageSize > 10000 {
		c.PageSize = 10000
	}

	c.userPrincipalName = CalculateUserPrincipalName(c.BindUsername, c.BaseDN)
	return nil
}

// Client for LDAP connection.
type Client struct {
	conn *ldap.Conn
	conf *Config
	refs map[string]*Client
}

// Dial creates a new Client and attempts to connect to the given LDAP server.
func Dial(conf *Config) (*Client, error) {
	if conf == nil {
		return nil, fmt.Errorf("Config cannot be nil")
	}

	if err := conf.Validate(); err != nil {
		return nil, err
	}

	cli := &Client{
		conf: conf,
		refs: map[string]*Client{},
	}

	if err := cli.Reconnect(); err != nil {
		return nil, err
	}

	return cli, nil
}

// Close the connection.
func (c *Client) Close() {
	c.conn.Close()

	for _, conn := range c.refs {
		conn.Close()
	}

	c.refs = map[string]*Client{}
}

// Reconnect to LDAP. This is used internally if the connection is interrupted.
func (c *Client) Reconnect() error {
	tlsConf := &tls.Config{
		InsecureSkipVerify: c.conf.SkipVerify,
	}

	conn, err := ldap.DialURL(c.conf.Address, ldap.DialWithTLSConfig(tlsConf))
	if err != nil {
		return fmt.Errorf("connecting to LDAP: %v", err)
	}

	if c.conf.StartTLS {
		if err := conn.StartTLS(tlsConf); err != nil {
			return fmt.Errorf("starting TLS: %v", err)
		}
	}

	if len(c.conf.BindPassword) == 0 {
		if err := conn.UnauthenticatedBind(c.conf.userPrincipalName); err != nil {
			return fmt.Errorf("unauthenticated bind to LDAP: %v", err)
		}
	} else if len(c.conf.userPrincipalName) > 0 && len(c.conf.BindPassword) > 0 {
		if err := conn.Bind(c.conf.userPrincipalName, c.conf.BindPassword); err != nil {
			return fmt.Errorf("binding to LDAP: %v", err)
		}
	}

	if c.conn != nil {
		c.conn.Close()
	}

	c.conn = conn
	return nil
}

// Config returns the Config object being used.
func (c *Client) Config() *Config {
	return c.conf
}

// Bind will attempt to bind as the given DN and password.
func (c *Client) Bind(username, password string) error {
	upn := CalculateUserPrincipalName(username, c.conf.BaseDN)

	if len(password) == 0 {
		if err := c.conn.UnauthenticatedBind(upn); err != nil {
			return fmt.Errorf("unauthenticated bind to LDAP: %v", err)
		}

		c.conf.BindUsername = username
		c.conf.userPrincipalName = upn
	} else if len(upn) > 0 && len(password) > 0 {
		if err := c.conn.Bind(upn, password); err != nil {
			return fmt.Errorf("binding to LDAP: %v", err)
		}

		c.conf.BindUsername = username
		c.conf.userPrincipalName = upn
		c.conf.BindPassword = password
	} else {
		return fmt.Errorf("cannot bind without a username")
	}

	// clear out any existing referral connections using previous credentials
	for _, conn := range c.refs {
		conn.Close()
	}

	c.refs = map[string]*Client{}
	return nil
}

// NewSearchRequest returns a new ldap.SearchRequest object with some defaults set.
func (c *Client) NewSearchRequest(filter string, attributes []string) *ldap.SearchRequest {
	return ldap.NewSearchRequest(
		c.conf.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		c.conf.DefaultTimeLimit,
		false,
		filter,
		attributes,
		nil,
	)
}

// Search is the low-level method of searching LDAP, and returns SearchResult.
// This method automaticaly reconnects to LDAP and retries if there is a connection error.
func (c *Client) Search(req *ldap.SearchRequest) (*ldap.SearchResult, error) {
	resp, err := c.conn.SearchWithPaging(req, uint32(c.conf.PageSize))
	if IsErrConnectionClosed(err) {
		if err := c.Reconnect(); err != nil {
			return nil, fmt.Errorf("while attempting to reconnect: %v", err)
		}

		return c.conn.SearchWithPaging(req, uint32(c.conf.PageSize))
	}

	if c.conf.FollowReferrals {
		origBaseDN := req.BaseDN
		c.configureReferrals(resp.Referrals)
		for _, ref := range resp.Referrals {
			conn := c.refs[ref]
			if conn == nil {
				continue
			}

			// change the initial base DN to the DN indicated by the referral
			req.BaseDN = conn.conf.BaseDN

			refResp, err := conn.Search(req)
			if err != nil {
				log.Printf("could not follow referral: %s: %v\n", ref, err)
				continue
			}

			if len(refResp.Entries) > 0 {
				resp.Entries = append(resp.Entries, refResp.Entries...)
			}
		}
		req.BaseDN = origBaseDN
	}

	return resp, err
}

// Add adds a new entry to the directory.
func (c *Client) Add(req *ldap.AddRequest) error {
	err := c.conn.Add(req)
	if IsErrConnectionClosed(err) {
		if err := c.Reconnect(); err != nil {
			return fmt.Errorf("while attempting to reconnect: %v", err)
		}

		return c.conn.Add(req)
	}

	return err
}

// Modify an existing entry.
func (c *Client) Modify(req *ldap.ModifyRequest) error {
	err := c.conn.Modify(req)
	if IsErrConnectionClosed(err) {
		if err := c.Reconnect(); err != nil {
			return fmt.Errorf("while attempting to reconnect: %v", err)
		}

		return c.conn.Modify(req)
	}

	return err
}

// Delete an existing entry.
func (c *Client) Delete(req *ldap.DelRequest) error {
	err := c.conn.Del(req)
	if IsErrConnectionClosed(err) {
		if err := c.Reconnect(); err != nil {
			return fmt.Errorf("while attempting to reconnect: %v", err)
		}

		return c.conn.Del(req)
	}

	return err
}

// SetPassword sets the password for a user.
func (c *Client) SetPassword(userDN string, password string) error {
	encodedPW, err := formatPassword(password)
	if err != nil {
		return fmt.Errorf("encoding password: %v", err)
	}

	req := &ldap.ModifyRequest{
		DN: userDN,
		Changes: []ldap.Change{
			{
				Operation: ldap.ReplaceAttribute,
				Modification: ldap.PartialAttribute{
					Type: AttributeUnicodePassword,
					Vals: []string{encodedPW},
				},
			},
		},
	}

	return c.conn.Modify(req)
}

// configureReferrals configures suggested referral clients.
func (c *Client) configureReferrals(referrals []string) {
	for _, ref := range referrals {
		if c.refs[ref] == nil {
			u, err := url.Parse(ref)
			if err != nil {
				log.Printf("cannot parse referral: %s: %v\n", ref, err)
				continue
			}

			conf := &Config{
				Address:          fmt.Sprintf("%s://%s", u.Scheme, u.Host),
				BaseDN:           strings.TrimSuffix(strings.TrimPrefix(u.Path, "/"), "/"),
				BindUsername:     c.conf.BindUsername,
				BindPassword:     c.conf.BindPassword,
				DefaultTimeLimit: c.conf.DefaultTimeLimit,
				FollowReferrals:  false,
				PageSize:         c.conf.PageSize,
				SkipVerify:       c.conf.SkipVerify,
				StartTLS:         c.conf.StartTLS,
			}

			conn, err := Dial(conf)
			if err != nil {
				log.Printf("dialing referral failed: %s: %v\n", ref, err)
				continue
			}

			c.refs[ref] = conn
		}
	}
}

// IsErrConnectionClosed determines if the given error is a connection closed message.
// This is used interally to determine if a reconnect is required.
func IsErrConnectionClosed(err error) bool {
	return err != nil && strings.Contains(err.Error(), "connection closed")
}

// IsDNSanitized determines if the given DN is sanitized to prevent LDAP injection.
func IsDNSanitized(dn string) bool {
	// See http://tools.ietf.org/search/rfc4515
	badCharacters := "\x00()*\\"
	return !strings.ContainsAny(dn, badCharacters)
}

// IsNameSanitized determines if the given name is sanitized to prevent LDAP injection.
func IsNameSanitized(name string) bool {
	// See http://tools.ietf.org/search/rfc4514: "special characters"
	badCharacters := "\x00()*\\,='\"#+;<>"
	return !strings.ContainsAny(name, badCharacters)
}

// ParseBaseDN returns only the base portion of a DN.
func ParseBaseDN(dn string) string {
	if len(dn) < 3 {
		return dn
	}

	if strings.EqualFold(dn[:3], "dc=") {
		return dn
	}

	dnLower := strings.ToLower(dn)
	idx := strings.Index(dnLower, ",dc=")
	if idx == -1 {
		return dn
	}

	return dn[idx+1:]
}

// ParseDomainFromDN parses the domain in dot notation from a DN.
func ParseDomainFromDN(dn string) string {
	if len(dn) < 3 {
		return dn
	}

	dn = strings.ToLower(ParseBaseDN(dn))
	if dn[:3] != "dc=" {
		return dn
	}

	return strings.ReplaceAll(dn[3:], ",dc=", ".")
}

// ParseBaseDNFromDomain parses the base DN from domain dot notation.
func ParseBaseDNFromDomain(domain string) string {
	if len(domain) == 0 {
		return domain
	}

	if idx := strings.Index(domain, "://"); idx > -1 {
		domain = domain[idx+3:]
	}

	domain = strings.SplitN(domain, ":", 2)[0]
	domain = strings.SplitN(domain, "/", 2)[0]

	return fmt.Sprintf("dc=%s", strings.ReplaceAll(domain, ".", ",dc="))
}

// CalculateUserPrincipalName attempts to calculate the userPrincipalDomain of the given username.
// If the username contains the `@` or `=` symbols, it is returned as-is. Otherwise, the domain is calculated
// based on the BaseDN.
func CalculateUserPrincipalName(username, baseDN string) string {
	if len(username) == 0 || len(baseDN) == 0 || strings.Contains(username, "@") || strings.Contains(username, "=") {
		return username
	}

	domain := ParseDomainFromDN(baseDN)
	return fmt.Sprintf("%s@%s", username, domain)
}

// formatPassword to utf16 and wrap in double quotes.
func formatPassword(password string) (string, error) {
	utf16 := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	return utf16.NewEncoder().String(fmt.Sprintf(`"%s"`, password))
}
