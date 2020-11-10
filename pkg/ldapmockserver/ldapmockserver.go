package ldapmockserver

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/deejross/direktor/pkg/ldapcli"
	"github.com/lor00x/goldap/message"
	"github.com/vjeantet/ldapserver"
)

const (
	// TestBindDN is the configured BindUsername for binding using a full DN.
	TestBindDN = "cn=read-only-admin,dc=example,dc=com"
	// TestBindUN is the configured BindUsername for binding.
	TestBindUN = "read-only-admin"
	// TestBindPW is the configured password for the BindDN for binding.
	TestBindPW = "password"
	// TestBaseDN is the configure base DN for the directory.
	TestBaseDN = "dc=example,dc=com"
)

var directory = []map[string]string{
	{
		ldapcli.AttributeDistinguishedName: "cn=unit-tester,ou=generic-ids,dc=example,dc=com",
		ldapcli.AttributeCommonName:        "unit-tester",
		ldapcli.AttributeDisplayName:       "Unit Tester",
		ldapcli.AttributeDepartment:        "Generic IDs",
		ldapcli.AttributeMail:              "",
		ldapcli.AttributeUserPrincipalName: "unit-tester@example.com",
		ldapcli.AttributeObjectClass:       ldapcli.ObjectClassPerson,
	},
	{
		ldapcli.AttributeDistinguishedName: "cn=einstein,ou=scientists,dc=example,dc=com",
		ldapcli.AttributeCommonName:        "einstein",
		ldapcli.AttributeDisplayName:       "Albert Einstein",
		ldapcli.AttributeDepartment:        "Scientists",
		ldapcli.AttributeMail:              "einstein@example.com",
		ldapcli.AttributeUserPrincipalName: "einstein@example.com",
		ldapcli.AttributeObjectClass:       ldapcli.ObjectClassPerson,
	},
	{
		ldapcli.AttributeDistinguishedName: "cn=newton,ou=scientists,dc=example,dc=com",
		ldapcli.AttributeCommonName:        "newton",
		ldapcli.AttributeDisplayName:       "Isaac Newton",
		ldapcli.AttributeDepartment:        "Scientists",
		ldapcli.AttributeMail:              "newton@example.com",
		ldapcli.AttributeUserPrincipalName: "newton@example.com",
		ldapcli.AttributeObjectClass:       ldapcli.ObjectClassPerson,
	},
	{
		ldapcli.AttributeDistinguishedName: "cn=tesla,ou=scientists,dc=example,dc=com",
		ldapcli.AttributeCommonName:        "tesla",
		ldapcli.AttributeDisplayName:       "Nikola Tesla",
		ldapcli.AttributeDepartment:        "Scientists",
		ldapcli.AttributeMail:              "tesla@example.com",
		ldapcli.AttributeUserPrincipalName: "tesla@example.com",
		ldapcli.AttributeObjectClass:       ldapcli.ObjectClassPerson,
	},
}

// Start the mock LDAP server.
func Start(addr string) (chan struct{}, error) {
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

	stopCh := make(chan struct{})
	go func() {
		<-stopCh
		server.Stop()
	}()

	errCh := make(chan error)
	go func() {
		if err := server.ListenAndServe(addr); err != nil {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return nil, err
	case <-time.After(time.Second):
	}

	return stopCh, nil
}

// Size gets the number of entries in the directory.
func Size() int {
	return len(directory)
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
		if string(req.Name()) == TestBindDN && req.AuthenticationSimple().String() == TestBindPW {
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
		ldapcli.AttributeDistinguishedName: string(req.Entry()),
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
		if m[ldapcli.AttributeDistinguishedName] == string(req.Object()) {
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
		if m[ldapcli.AttributeDistinguishedName] == req {
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
		if !strings.HasSuffix(m[ldapcli.AttributeDistinguishedName], string(req.BaseObject())) {
			continue
		}

		if !entryMatchesFilter(m, req.Filter()) {
			continue
		}

		e := ldapserver.NewSearchResultEntry(m[ldapcli.AttributeDistinguishedName])
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
