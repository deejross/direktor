package ldapcli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Integration tests have been moved to the ldapmockserver package.

func TestParseBaseDN(t *testing.T) {
	require.Equal(t, "dc=server,dc=local", ParseBaseDN("dc=server,dc=local"))
	require.Equal(t, "DC=server,DC=local", ParseBaseDN("DC=server,DC=local"))
	require.Equal(t, "dc=server,dc=local", ParseBaseDN("cn=tesla,ou=Users,dc=server,dc=local"))
	require.Equal(t, "DC=server,DC=local", ParseBaseDN("CN=tesla,OU=Users,DC=server,DC=local"))
	require.Equal(t, "invalid", ParseBaseDN("invalid"))
	require.Equal(t, "", ParseBaseDN(""))
}

func TestParseDomainFromDN(t *testing.T) {
	require.Equal(t, "server.local", ParseDomainFromDN("dc=server,dc=local"))
	require.Equal(t, "server.local", ParseDomainFromDN("DC=server,DC=local"))
	require.Equal(t, "server.local", ParseDomainFromDN("cn=tesla,ou=Users,dc=server,dc=local"))
	require.Equal(t, "server.local", ParseDomainFromDN("CN=tesla,OU=Users,DC=server,DC=local"))
	require.Equal(t, "invalid", ParseDomainFromDN("invalid"))
	require.Equal(t, "", ParseDomainFromDN(""))
}

func TestParseBaseDNFromDomain(t *testing.T) {
	require.Equal(t, "dc=server,dc=local", ParseBaseDNFromDomain("server.local"))
	require.Equal(t, "dc=server,dc=local", ParseBaseDNFromDomain("http://server.local"))
	require.Equal(t, "dc=server,dc=local", ParseBaseDNFromDomain("ldap://server.local:389"))
	require.Equal(t, "dc=server,dc=local", ParseBaseDNFromDomain("ldaps://server.local/somepath"))
	require.Equal(t, "dc=invalid", ParseBaseDNFromDomain("invalid"))
	require.Equal(t, "", ParseBaseDNFromDomain(""))
}
