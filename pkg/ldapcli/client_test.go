package ldapcli

import (
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/lor00x/goldap/message"
	"github.com/stretchr/testify/require"
	"github.com/vjeantet/ldapserver"
)

const (
	testAddress = "ldap://127.0.0.1:10389"
	testBindDN  = "cn=read-only-admin,dc=example,dc=com"
	testBindPW  = "password"
	testBaseDN  = "dc=example,dc=com"
)

var cli *Client
var directory = []map[string]string{
	{
		AttributeDistinguishedName: "cn=einstein,ou=scientists,dc=example,dc=com",
		AttributeCommonName:        "einstein",
		AttributeDisplayName:       "Albert Einstein",
		AttributeDepartment:        "Scientists",
		AttributeMail:              "einstein@example.com",
		AttributeObjectClass:       ObjectClassPerson,
	},
	{
		AttributeDistinguishedName: "cn=newton,ou=scientists,dc=example,dc=com",
		AttributeCommonName:        "newton",
		AttributeDisplayName:       "Isaac Newton",
		AttributeDepartment:        "Scientists",
		AttributeMail:              "newton@example.com",
		AttributeObjectClass:       ObjectClassPerson,
	},
	{
		AttributeDistinguishedName: "cn=tesla,ou=scientists,dc=example,dc=com",
		AttributeCommonName:        "tesla",
		AttributeDisplayName:       "Nikola Tesla",
		AttributeDepartment:        "Scientists",
		AttributeMail:              "tesla@example.com",
		AttributeObjectClass:       ObjectClassPerson,
	},
}

func TestMain(m *testing.M) {
	// start the LDAP server
	ldapserver.Logger = log.New(os.NewFile(0, os.DevNull), "", log.LstdFlags)
	server := ldapserver.NewServer()
	routes := ldapserver.NewRouteMux()
	routes.NotFound(handleNotFound)
	routes.Abandon(handleAbandon)
	routes.Bind(handleBind)
	routes.Add(handleAdd)
	routes.Modify(handleModify)
	routes.Delete(handleDelete)
	routes.Search(handleSearch)

	server.Handle(routes)
	go server.ListenAndServe("127.0.0.1:10389")
	time.Sleep(time.Second)

	// create the LDAP client connection
	conf := NewConfig(testAddress, testBaseDN)
	conf.BindDN = testBindDN
	conf.BindPassword = testBindPW

	var err error
	cli, err = Dial(conf)
	if err != nil {
		log.Fatalln(err)
	}

	code := m.Run()
	cli.Close()
	server.Stop()
	os.Exit(code)
}

func TestSearchAll(t *testing.T) {
	req := cli.NewSearchRequest(`(cn=*)`, []string{AttributeCommonName, AttributeDisplayName})
	resp, err := cli.Search(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, resp.Referrals)
	require.Len(t, resp.Entries, len(directory))
}

func TestSearchNotFound(t *testing.T) {
	require.NotNil(t, cli)

	req := cli.NewSearchRequest(`(&(objectClass=person)(cn=unknown))`, []string{AttributeCommonName, AttributeDisplayName})
	resp, err := cli.Search(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, resp.Referrals)
	require.Empty(t, resp.Entries)
}

func TestSearchByCN(t *testing.T) {
	require.NotNil(t, cli)

	req := cli.NewSearchRequest(`(&(objectClass=person)(cn=tesla))`, []string{AttributeCommonName, AttributeDisplayName})
	resp, err := cli.Search(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, resp.Referrals)
	require.Len(t, resp.Entries, 1)
	require.Equal(t, "cn=tesla,ou=scientists,dc=example,dc=com", resp.Entries[0].DN)
	require.Equal(t, "tesla", resp.Entries[0].GetAttributeValue(AttributeCommonName))
	require.Equal(t, "Nikola Tesla", resp.Entries[0].GetAttributeValue(AttributeDisplayName))
	require.Empty(t, resp.Entries[0].GetAttributeValue(AttributeMail))
	require.Empty(t, resp.Entries[0].GetAttributeValue(AttributeDepartment))
	require.Empty(t, resp.Entries[0].GetAttributeValue(AttributeObjectClass))
}

func TestSearchByDN(t *testing.T) {
	require.NotNil(t, cli)

	req := cli.NewSearchRequest(`(distinguishedName=cn=newton,ou=scientists,dc=example,dc=com)`, []string{AttributeCommonName, AttributeDisplayName, AttributeMail})
	resp, err := cli.Search(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, resp.Referrals)
	require.Len(t, resp.Entries, 1)
	require.Equal(t, "cn=newton,ou=scientists,dc=example,dc=com", resp.Entries[0].DN)
	require.Equal(t, "newton", resp.Entries[0].GetAttributeValue(AttributeCommonName))
	require.Equal(t, "Isaac Newton", resp.Entries[0].GetAttributeValue(AttributeDisplayName))
	require.Equal(t, "newton@example.com", resp.Entries[0].GetAttributeValue(AttributeMail))
	require.Empty(t, resp.Entries[0].GetAttributeValue(AttributeDepartment))
	require.Empty(t, resp.Entries[0].GetAttributeValue(AttributeObjectClass))
}

func TestAdd(t *testing.T) {
	require.NotNil(t, cli)

	addReq := &ldap.AddRequest{
		DN: "cn=washington,ou=presidents,dc=example,dc=com",
		Attributes: []ldap.Attribute{
			{
				Type: AttributeCommonName,
				Vals: []string{"washington"},
			},
			{
				Type: AttributeDepartment,
				Vals: []string{"scientists"}, // this is wrong on purpose, we'll change it later
			},
			{
				Type: AttributeDisplayName,
				Vals: []string{"George Washington"},
			},
			{
				Type: AttributeMail,
				Vals: []string{"washington@example.com"},
			},
			{
				Type: AttributeObjectClass,
				Vals: []string{"person"},
			},
		},
	}

	require.NoError(t, cli.Add(addReq))

	req := cli.NewSearchRequest(`(&(objectClass=person)(cn=washington))`, []string{AttributeCommonName, AttributeDisplayName})
	resp, err := cli.Search(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, resp.Referrals)
	require.Len(t, resp.Entries, 1)
	require.Equal(t, "cn=washington,ou=presidents,dc=example,dc=com", resp.Entries[0].DN)
	require.Equal(t, "washington", resp.Entries[0].GetAttributeValue(AttributeCommonName))
	require.Equal(t, "George Washington", resp.Entries[0].GetAttributeValue(AttributeDisplayName))
}

func TestModify(t *testing.T) {
	require.NotNil(t, cli)

	modReq := &ldap.ModifyRequest{
		DN: "cn=washington,ou=presidents,dc=example,dc=com",
		Changes: []ldap.Change{
			{
				Operation: ldap.ReplaceAttribute,
				Modification: ldap.PartialAttribute{
					Type: AttributeDepartment,
					Vals: []string{"presidents"},
				},
			},
		},
	}

	require.NoError(t, cli.Modify(modReq))

	req := cli.NewSearchRequest(`(&(objectClass=person)(cn=washington))`, []string{AttributeCommonName, AttributeDisplayName, AttributeDepartment})
	resp, err := cli.Search(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, resp.Referrals)
	require.Len(t, resp.Entries, 1)
	require.Equal(t, "cn=washington,ou=presidents,dc=example,dc=com", resp.Entries[0].DN)
	require.Equal(t, "washington", resp.Entries[0].GetAttributeValue(AttributeCommonName))
	require.Equal(t, "George Washington", resp.Entries[0].GetAttributeValue(AttributeDisplayName))
	require.Equal(t, "presidents", resp.Entries[0].GetAttributeValue(AttributeDepartment))
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

	req := cli.NewSearchRequest(`(&(objectClass=person)(cn=washington))`, []string{AttributeCommonName, AttributeDisplayName})
	resp, err := cli.Search(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, resp.Referrals)
	require.Empty(t, resp.Entries)
}

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

func handleNotFound(w ldapserver.ResponseWriter, m *ldapserver.Message) {
	switch m.ProtocolOpType() {
	case ldapserver.ApplicationBindRequest:
		resp := ldapserver.NewBindResponse(ldapserver.LDAPResultSuccess)
		w.Write(resp)
	default:
		resp := ldapserver.NewResponse(ldapserver.LDAPResultUnwillingToPerform)
		w.Write(resp)
	}
}

func handleAbandon(w ldapserver.ResponseWriter, m *ldapserver.Message) {
	req := m.GetAbandonRequest()
	if req, ok := m.Client.GetMessageByID(int(req)); ok {
		req.Abandon()
	}
}

func handleBind(w ldapserver.ResponseWriter, m *ldapserver.Message) {
	req := m.GetBindRequest()
	resp := ldapserver.NewBindResponse(ldapserver.LDAPResultSuccess)

	if req.AuthenticationChoice() == "simple" {
		if string(req.Name()) == testBindDN && req.AuthenticationSimple().String() == testBindPW {
			w.Write(resp)
			return
		}

		resp.SetResultCode(ldapserver.LDAPResultInvalidCredentials)
	} else {
		resp.SetResultCode(ldapserver.LDAPResultUnwillingToPerform)
	}

	w.Write(resp)
}

func handleAdd(w ldapserver.ResponseWriter, m *ldapserver.Message) {
	req := m.GetAddRequest()

	mp := map[string]string{
		AttributeDistinguishedName: string(req.Entry()),
	}

	for _, attr := range req.Attributes() {
		if len(attr.Vals()) == 0 {
			continue
		}

		mp[string(attr.Type_())] = string(attr.Vals()[0])
	}

	directory = append(directory, mp)

	resp := ldapserver.NewAddResponse(ldapserver.LDAPResultSuccess)
	w.Write(resp)
}

func handleModify(w ldapserver.ResponseWriter, m *ldapserver.Message) {
	req := m.GetModifyRequest()

	var mp map[string]string

	for _, m := range directory {
		if m[AttributeDistinguishedName] == string(req.Object()) {
			mp = m
			break
		}
	}

	if mp == nil {
		resp := ldapserver.NewModifyResponse(ldapserver.LDAPResultNoSuchObject)
		w.Write(resp)
		return
	}

	for _, change := range req.Changes() {
		mod := change.Modification()
		if len(mod.Vals()) == 0 {
			continue
		}

		switch change.Operation() {
		case ldapserver.ModifyRequestChangeOperationAdd, ldapserver.ModifyRequestChangeOperationReplace:
			mp[string(mod.Type_())] = string(mod.Vals()[0])
		case ldapserver.ModifyRequestChangeOperationDelete:
			delete(mp, string(mod.Type_()))
		}
	}

	resp := ldapserver.NewModifyResponse(ldapserver.LDAPResultSuccess)
	w.Write(resp)
}

func handleDelete(w ldapserver.ResponseWriter, m *ldapserver.Message) {
	req := string(m.GetDeleteRequest())

	for i, m := range directory {
		if m[AttributeDistinguishedName] == req {
			directory = append(directory[:i], directory[i+1:]...)

			resp := ldapserver.NewDeleteResponse(ldapserver.LDAPResultSuccess)
			w.Write(resp)
			return
		}
	}

	resp := ldapserver.NewDeleteResponse(ldapserver.LDAPResultNoSuchObject)
	w.Write(resp)
}

func handleSearch(w ldapserver.ResponseWriter, m *ldapserver.Message) {
	req := m.GetSearchRequest()

	for _, m := range directory {
		if !strings.HasSuffix(m[AttributeDistinguishedName], string(req.BaseObject())) {
			continue
		}

		if !entryMatchesFilter(m, req.Filter()) {
			continue
		}

		e := ldapserver.NewSearchResultEntry(m[AttributeDistinguishedName])
		for _, attr := range req.Attributes() {
			e.AddAttribute(message.AttributeDescription(attr), message.AttributeValue(m[string(attr)]))
		}

		w.Write(e)
	}

	w.Write(ldapserver.NewSearchResultDoneResponse(ldapserver.LDAPResultSuccess))
}

func entryMatchesFilter(m map[string]string, filter message.Filter) bool {
	switch f := filter.(type) {
	case message.FilterAnd:
		for _, child := range f {
			if !entryMatchesFilter(m, child) {
				return false
			}
		}
		return true
	case message.FilterOr:
		oneTrue := false
		for _, child := range f {
			if entryMatchesFilter(m, child) {
				oneTrue = true
			}
		}
		return oneTrue
	case message.FilterNot:
		return !entryMatchesFilter(m, f.Filter)
	case message.FilterSubstrings:
		for _, ss := range f.Substrings() {
			switch ssv := ss.(type) {
			case message.SubstringInitial:
				val := m[string(f.Type_())]
				return strings.HasPrefix(val, string(ssv))
			case message.SubstringFinal:
				val := m[string(f.Type_())]
				return strings.HasSuffix(val, string(ssv))
			case message.SubstringAny:
				val := m[string(f.Type_())]
				return strings.Contains(val, string(ssv))
			}
		}
	case message.FilterEqualityMatch:
		val := m[string(f.AttributeDesc())]
		return val == string(f.AssertionValue())
	case message.FilterGreaterOrEqual:
		val := m[string(f.AttributeDesc())]
		valF, _ := strconv.ParseFloat(val, 64)
		compareF, _ := strconv.ParseFloat(string(f.AssertionValue()), 64)

		return valF >= compareF
	case message.FilterLessOrEqual:
		val := m[string(f.AttributeDesc())]
		valF, _ := strconv.ParseFloat(val, 64)
		compareF, _ := strconv.ParseFloat(string(f.AssertionValue()), 64)

		return valF <= compareF
	case message.FilterPresent:
		_, ok := m[string(f)]
		return ok
	case message.FilterApproxMatch:
		val := m[string(f.AttributeDesc())]
		return strings.Contains(val, string(f.AssertionValue()))
	}

	return false
}
