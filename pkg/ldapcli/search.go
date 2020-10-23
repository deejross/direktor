package ldapcli

import (
	"fmt"
	"log"

	"github.com/go-ldap/ldap/v3"
)

// GroupMembers returns a list of members of the given group using a memberOf search.
// This does not return members of nested groups.
// Optionally, a list of attributes for the members can also be returned.
// If attributes is empty, only the objectClass attribute is returned.
func (c *Client) GroupMembers(groupDN string, attributes ...string) (*ldap.SearchResult, error) {
	// perform a (memberOf=groupDN) search with desired attributes
	filter := fmt.Sprintf(`(%s=%s)`, AttributeMemberOf, groupDN)
	if attributes == nil || len(attributes) == 0 {
		attributes = []string{AttributeObjectClass}
	}

	req := c.NewSearchRequest(filter, attributes)
	return c.Search(req)
}

// GroupMembersExtended gets members from a call to GroupMembers, then attempts
// to discover members from other domains that wouldn't otherwise be listed by GroupMembers.
// This is done by querying the group's `member` attribute and performing additional searches
// to retreive the requested attributes for any newly discovered members.
func (c *Client) GroupMembersExtended(groupDN string, attributes ...string) (*ldap.SearchResult, error) {
	// call GroupMembers
	if attributes == nil || len(attributes) == 0 {
		attributes = []string{AttributeObjectClass}
	}

	resp, err := c.GroupMembers(groupDN, attributes...)
	if err != nil {
		return resp, err
	}

	// index results to prevent duplicates
	index := map[string]struct{}{}
	for _, e := range resp.Entries {
		index[e.DN] = struct{}{}
	}

	// retreive the group's `member` attribute
	filter := fmt.Sprintf(`(%s=%s)`, AttributeDistinguishedName, groupDN)
	memberRange := "member;range=0-*"
	groupAttrs := []string{memberRange}
	req := c.NewSearchRequest(filter, groupAttrs)

	exResp, err := c.Search(req)
	if err != nil {
		return resp, err
	}

	if len(exResp.Entries) != 1 {
		return resp, nil
	}

	// ignore indexed members, for new members search for desired attributes and append to results
	members := exResp.Entries[0].GetAttributeValues(memberRange)
	for _, dn := range members {
		if _, ok := index[dn]; !ok {
			filter := fmt.Sprintf(`(%s=%s)`, AttributeDistinguishedName, dn)
			req := c.NewSearchRequest(filter, attributes)

			memResp, err := c.Search(req)
			if err != nil {
				log.Printf("could not get member attributes: %s: %v", dn, err)
				resp.Entries = append(resp.Entries, &ldap.Entry{DN: dn, Attributes: []*ldap.EntryAttribute{}})
				continue
			}

			if len(memResp.Entries) > 0 {
				resp.Entries = append(resp.Entries, memResp.Entries[0])
			}
		}
	}

	return resp, nil
}

// OrganizationalUnitMembers returns a list of members of the given organizational
// unit. The baseDN argument, if an empty string, will default to configured baseDN.
// Optionally, a list of attributes for the members can also be returned. If
// attributes is empty, only the distinguishedName attribute is requred.
func (c *Client) OrganizationalUnitMembers(baseDN string, attributes ...string) (*ldap.SearchResult, error) {
	filter := fmt.Sprintf(`(objectClass=*)`)
	if attributes == nil {
		attributes = []string{}
	}

	req := c.NewSearchRequest(filter, attributes)
	req.Scope = ldap.ScopeSingleLevel
	if len(baseDN) > 0 {
		req.BaseDN = baseDN
	}

	return c.Search(req)
}
