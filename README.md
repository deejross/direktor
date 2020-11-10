# Direktor
A web-based GUI and CLI for viewing objects in LDAP and Active Directory.

## Features
* Works on Windows, Mac, and Linux
* Supports multiple domains
* Follows referrals for subdomains in a forest
* Discovers group membership across subdomains
* Command-line utility
* REST API
* Materials-based web UI

## Technology
The technology stack includes:
* Go (API)
* Vue (UI)

## Using the CLI
See the CLI help:
```bash
$ go run cmd/cli/main.go --help
direktorcli is used to search objects in LDAP/Active Directory

Usage:
  direktorcli [command]

Available Commands:
  help        Help about any command
  list        List members of an Organizational Unit
  login       Login creates a state file with login information for convenience.
  members     List members of a group
  search      Search directory

Flags:
  -a, --address string    Address to LDAP server in format: ldap://server.local:389 or ldaps://server.local:636
  -b, --basedn string     BaseDN for searching, defaults to auto discovery
  -h, --help              help for direktorcli
      --insecure          Skip TLS validation errors
  -p, --password string   Password to use for authentication, if not set you will be prompted
      --start-tls         Start TLS
  -u, --username string   Username to use for authentication

Use "direktorcli [command] --help" for more information about a command.
```

Example:
```bash
$ go run cmd/cli/main.go login -a ldap://127.0.0.1:10389 -b dc=example,dc=com -u read-only-admin

$ go run cmd/cli/main.go search --by-attr=samaccountname=myusername
Distinguished Name: CN=myusername,OU=Users,DC=example,DC=com
Attributes:
  objectClass: top; person; organizationalPerson; user
  cn: myusername
```

There are many options for searching and returning attributes. To learn more, use `go run cmd/cli/main.go search --help`

## Status
**EXPERIMENTAL** 

* [x] Stable LDAP client module
* [x] CLI utility
* [ ] Initial API with documentation
* [ ] Web-based UI

This project is still new and most of the functionality doesn't exist yet. The Go LDAP client and CLI are working well and documented. The API is under development. The UI is just a skeleton at this point. Once the API is stablized, work on the UI can begin.
