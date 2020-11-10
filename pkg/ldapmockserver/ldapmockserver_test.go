package ldapmockserver

import (
	"log"
	"os"
	"testing"

	"github.com/deejross/direktor/pkg/ldapcli"
	"github.com/go-ldap/ldap/v3"
	"github.com/stretchr/testify/require"
)

const (
	testAddress = "ldap://127.0.0.1:10389"
)

var cli *ldapcli.Client

func TestMain(m *testing.M) {
	// start the LDAP server
	stopCh, err := Start("127.0.0.1:10389")
	if err != nil {
		log.Fatalln(err)
	}

	// create the LDAP client connection
	conf := ldapcli.NewConfig(testAddress, TestBaseDN)
	conf.BindUsername = TestBindDN
	conf.BindPassword = TestBindPW

	cli, err = ldapcli.Dial(conf)
	if err != nil {
		log.Fatalln(err)
	}

	code := m.Run()
	cli.Close()
	stopCh <- struct{}{}
	os.Exit(code)
}

func TestSearchAll(t *testing.T) {
	req := cli.NewSearchRequest(`(cn=*)`, []string{ldapcli.AttributeCommonName, ldapcli.AttributeDisplayName})
	resp, err := cli.Search(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, resp.Referrals)
	require.Len(t, resp.Entries, Size())
}

func TestSearchNotFound(t *testing.T) {
	require.NotNil(t, cli)

	req := cli.NewSearchRequest(`(&(objectClass=person)(cn=unknown))`, []string{ldapcli.AttributeCommonName, ldapcli.AttributeDisplayName})
	resp, err := cli.Search(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, resp.Referrals)
	require.Empty(t, resp.Entries)
}

func TestSearchByCN(t *testing.T) {
	require.NotNil(t, cli)

	req := cli.NewSearchRequest(`(&(objectClass=person)(cn=tesla))`, []string{ldapcli.AttributeCommonName, ldapcli.AttributeDisplayName})
	resp, err := cli.Search(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, resp.Referrals)
	require.Len(t, resp.Entries, 1)
	require.Equal(t, "cn=tesla,ou=scientists,dc=example,dc=com", resp.Entries[0].DN)
	require.Equal(t, "tesla", resp.Entries[0].GetAttributeValue(ldapcli.AttributeCommonName))
	require.Equal(t, "Nikola Tesla", resp.Entries[0].GetAttributeValue(ldapcli.AttributeDisplayName))
	require.Empty(t, resp.Entries[0].GetAttributeValue(ldapcli.AttributeMail))
	require.Empty(t, resp.Entries[0].GetAttributeValue(ldapcli.AttributeDepartment))
	require.Empty(t, resp.Entries[0].GetAttributeValue(ldapcli.AttributeObjectClass))
}

func TestSearchByDN(t *testing.T) {
	require.NotNil(t, cli)

	req := cli.NewSearchRequest(`(distinguishedName=cn=newton,ou=scientists,dc=example,dc=com)`, []string{ldapcli.AttributeCommonName, ldapcli.AttributeDisplayName, ldapcli.AttributeMail})
	resp, err := cli.Search(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, resp.Referrals)
	require.Len(t, resp.Entries, 1)
	require.Equal(t, "cn=newton,ou=scientists,dc=example,dc=com", resp.Entries[0].DN)
	require.Equal(t, "newton", resp.Entries[0].GetAttributeValue(ldapcli.AttributeCommonName))
	require.Equal(t, "Isaac Newton", resp.Entries[0].GetAttributeValue(ldapcli.AttributeDisplayName))
	require.Equal(t, "newton@example.com", resp.Entries[0].GetAttributeValue(ldapcli.AttributeMail))
	require.Empty(t, resp.Entries[0].GetAttributeValue(ldapcli.AttributeDepartment))
	require.Empty(t, resp.Entries[0].GetAttributeValue(ldapcli.AttributeObjectClass))
}

func TestAdd(t *testing.T) {
	require.NotNil(t, cli)

	addReq := &ldap.AddRequest{
		DN: "cn=washington,ou=presidents,dc=example,dc=com",
		Attributes: []ldap.Attribute{
			{
				Type: ldapcli.AttributeCommonName,
				Vals: []string{"washington"},
			},
			{
				Type: ldapcli.AttributeDepartment,
				Vals: []string{"scientists"}, // this is wrong on purpose, we'll change it later
			},
			{
				Type: ldapcli.AttributeDisplayName,
				Vals: []string{"George Washington"},
			},
			{
				Type: ldapcli.AttributeMail,
				Vals: []string{"washington@example.com"},
			},
			{
				Type: ldapcli.AttributeObjectClass,
				Vals: []string{"person"},
			},
		},
	}

	require.NoError(t, cli.Add(addReq))

	req := cli.NewSearchRequest(`(&(objectClass=person)(cn=washington))`, []string{ldapcli.AttributeCommonName, ldapcli.AttributeDisplayName})
	resp, err := cli.Search(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, resp.Referrals)
	require.Len(t, resp.Entries, 1)
	require.Equal(t, "cn=washington,ou=presidents,dc=example,dc=com", resp.Entries[0].DN)
	require.Equal(t, "washington", resp.Entries[0].GetAttributeValue(ldapcli.AttributeCommonName))
	require.Equal(t, "George Washington", resp.Entries[0].GetAttributeValue(ldapcli.AttributeDisplayName))
}

func TestModify(t *testing.T) {
	require.NotNil(t, cli)

	modReq := &ldap.ModifyRequest{
		DN: "cn=washington,ou=presidents,dc=example,dc=com",
		Changes: []ldap.Change{
			{
				Operation: ldap.ReplaceAttribute,
				Modification: ldap.PartialAttribute{
					Type: ldapcli.AttributeDepartment,
					Vals: []string{"presidents"},
				},
			},
		},
	}

	require.NoError(t, cli.Modify(modReq))

	req := cli.NewSearchRequest(`(&(objectClass=person)(cn=washington))`, []string{ldapcli.AttributeCommonName, ldapcli.AttributeDisplayName, ldapcli.AttributeDepartment})
	resp, err := cli.Search(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, resp.Referrals)
	require.Len(t, resp.Entries, 1)
	require.Equal(t, "cn=washington,ou=presidents,dc=example,dc=com", resp.Entries[0].DN)
	require.Equal(t, "washington", resp.Entries[0].GetAttributeValue(ldapcli.AttributeCommonName))
	require.Equal(t, "George Washington", resp.Entries[0].GetAttributeValue(ldapcli.AttributeDisplayName))
	require.Equal(t, "presidents", resp.Entries[0].GetAttributeValue(ldapcli.AttributeDepartment))
}

func TestSetPassword(t *testing.T) {
	require.NotNil(t, cli)

	require.NoError(t, cli.SetPassword("cn=washington,ou=presidents,dc=example,dc=com", "super-secret"))
}

func TestDelete(t *testing.T) {
	require.NotNil(t, cli)

	delReq := &ldap.DelRequest{
		DN: "cn=washington,ou=presidents,dc=example,dc=com",
	}

	require.NoError(t, cli.Delete(delReq))

	req := cli.NewSearchRequest(`(&(objectClass=person)(cn=washington))`, []string{ldapcli.AttributeCommonName, ldapcli.AttributeDisplayName})
	resp, err := cli.Search(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, resp.Referrals)
	require.Empty(t, resp.Entries)
}
