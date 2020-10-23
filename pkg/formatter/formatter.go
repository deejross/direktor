package formatter

import (
	"encoding/json"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-ldap/ldap/v3"
	"github.com/go-ldap/ldif"
	"gopkg.in/yaml.v2"
)

// Formatters is a list of registered formatters.
var Formatters = map[string]Formatter{
	"json":        JSONFormatter,
	"json-pretty": JSONPrettyFormatter,
	"ldif":        LDIFFormatter,
	"text":        TextFormatter,
	"yaml":        YAMLFormatter,
}

// Formatter interface for outputing objects into multiple formats.
type Formatter func(v interface{}) ([]byte, error)

// JSONFormatter outputs minified JSON.
func JSONFormatter(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// JSONPrettyFormatter outputs human-readable JSON.
func JSONPrettyFormatter(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// LDIFFormatter outputs LDAP entries in LDIF, will return error for non-LDAP types.
func LDIFFormatter(v interface{}) ([]byte, error) {
	entries := []*ldif.Entry{}

	switch obj := v.(type) {
	case *ldap.SearchResult:
		for _, e := range obj.Entries {
			entries = append(entries, &ldif.Entry{Entry: e})
		}
	case []*ldap.Entry:
		for _, e := range obj {
			entries = append(entries, &ldif.Entry{Entry: e})
		}
	case *ldap.Entry:
		entries = append(entries, &ldif.Entry{Entry: obj})
	default:
		return nil, fmt.Errorf("object must be one of type: *ldap.SearchResult, []*ldap.Entry, *ldap.Entry")
	}

	ld := &ldif.LDIF{
		Entries: entries,
		Version: 1,
	}

	str, err := ldif.Marshal(ld)
	return []byte(str), err
}

// TextFormatter outputs human-readable text.
func TextFormatter(v interface{}) ([]byte, error) {
	return []byte(spew.Sdump(v)), nil
}

// YAMLFormatter outputs to YAML.
func YAMLFormatter(v interface{}) ([]byte, error) {
	return yaml.Marshal(v)
}

// Format the given object into the given format. If format is an empty string, `text` is used.
func Format(format string, v interface{}) ([]byte, error) {
	if len(format) == 0 {
		format = "text"
	}

	f := Formatters[format]
	if f == nil {
		return nil, fmt.Errorf("unrecognized format: %s", format)
	}

	return f(v)
}

func init() {
	spew.Config.Indent = "  "
	spew.Config.DisableCapacities = true
	spew.Config.DisablePointerAddresses = true
	spew.Config.SortKeys = true
	spew.Config.SpewKeys = true
}
