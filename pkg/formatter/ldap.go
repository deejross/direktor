package formatter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-ldap/ldap/v3"
	"github.com/go-ldap/ldif"
	"gopkg.in/yaml.v2"
)

// LDAPAttribute represents an LDAP attribute.
type LDAPAttribute struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

// LDAPEntry represents an LDAP entry.
type LDAPEntry struct {
	DistinguishedName string          `json:"distinguishedName"`
	Attributes        []LDAPAttribute `json:"attributes"`
}

// LDAPFormatters is a list of registered formatters.
var LDAPFormatters = map[string]LDAPFormatter{
	"json":        LDAPFormatterJSON,
	"json-pretty": LDAPFormatterJSONPretty,
	"ldif":        LDAPFormatterLDIF,
	"text":        LDAPFormatterText,
	"yaml":        LDAPFormatterYAML,
}

// LDAPFormatter interface for outputing objects into multiple formats.
type LDAPFormatter func(resp *ldap.SearchResult) ([]byte, error)

// LDAPFormatterJSON outputs minified JSON.
func LDAPFormatterJSON(resp *ldap.SearchResult) ([]byte, error) {
	entries := preprocessLDAPSearchResult(resp)
	return json.Marshal(entries)
}

// LDAPFormatterJSONPretty outputs human-readable JSON.
func LDAPFormatterJSONPretty(resp *ldap.SearchResult) ([]byte, error) {
	entries := preprocessLDAPSearchResult(resp)
	return json.MarshalIndent(entries, "", "  ")
}

// LDAPFormatterLDIF outputs LDAP entries in LDIF.
func LDAPFormatterLDIF(resp *ldap.SearchResult) ([]byte, error) {
	entries := []*ldif.Entry{}

	for _, e := range resp.Entries {
		entries = append(entries, &ldif.Entry{Entry: e})
	}

	ld := &ldif.LDIF{
		Entries: entries,
		Version: 1,
	}

	str, err := ldif.Marshal(ld)
	return []byte(str), err
}

// LDAPFormatterText outputs human-readable text.
func LDAPFormatterText(resp *ldap.SearchResult) ([]byte, error) {
	entries := preprocessLDAPSearchResult(resp)

	buf := bytes.Buffer{}

	for i, e := range entries {
		buf.WriteString(fmt.Sprintf("Distinguished Name: %s\n", e.DistinguishedName))
		buf.WriteString(fmt.Sprintln("Attributes:"))

		for _, attr := range e.Attributes {
			buf.WriteString(fmt.Sprintf("  %s: %s\n", attr.Name, strings.Join(attr.Values, "; ")))
		}

		// only add newline character if there are more entries to print
		if i < len(entries)-1 {
			buf.WriteString("\n")
		}
	}

	return buf.Bytes(), nil
}

// LDAPFormatterYAML outputs to YAML.
func LDAPFormatterYAML(resp *ldap.SearchResult) ([]byte, error) {
	entries := preprocessLDAPSearchResult(resp)
	return yaml.Marshal(entries)
}

// FormatLDAPSearchResult attempts to format an LDAP SearchResult it to the given format.
// If format is an empty string, `text` is used.
func FormatLDAPSearchResult(format string, resp *ldap.SearchResult) ([]byte, error) {
	if len(format) == 0 {
		format = "text"
	}

	f := LDAPFormatters[format]
	if f == nil {
		return nil, fmt.Errorf("unrecognized format: %s", format)
	}

	return f(resp)
}

func preprocessLDAPSearchResult(resp *ldap.SearchResult) []LDAPEntry {
	entries := []LDAPEntry{}

	for _, e := range resp.Entries {
		entry := LDAPEntry{
			DistinguishedName: e.DN,
			Attributes:        []LDAPAttribute{},
		}

		for _, attr := range e.Attributes {
			entry.Attributes = append(entry.Attributes, LDAPAttribute{
				Name:   attr.Name,
				Values: attr.Values,
			})
		}

		entries = append(entries, entry)
	}

	return entries
}

func init() {
	spew.Config.Indent = "  "
	spew.Config.DisableCapacities = true
	spew.Config.DisablePointerAddresses = true
	spew.Config.SortKeys = true
	spew.Config.SpewKeys = true
}
