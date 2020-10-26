package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/deejross/direktor/pkg/formatter"
	"github.com/deejross/direktor/pkg/ldapcli"
	"github.com/go-ldap/ldap/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	defaultConfigDir  = ""
	defaultConfigFile = ""
)

var rootCmd = &cobra.Command{
	Use:   "direktorcli",
	Short: "direktorcli is used to search objects in LDAP/Active Directory",
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login creates a state file with login information for convenience.",
	Run: func(cmd *cobra.Command, args []string) {
		cli := getClient(cmd)
		cli.Close()

		if err := viper.WriteConfig(); err != nil {
			fatal(fmt.Sprintf("unable to write state: %s\n", err))
		}
	},
}

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search directory",
	Run: func(cmd *cobra.Command, args []string) {
		cli := getClient(cmd)
		defer cli.Close()

		resp, err := search(cmd, cli)
		if err != nil {
			fatal(err.Error())
		}

		output, _ := cmd.Flags().GetString("output")
		b, err := formatter.FormatLDAPSearchResult(output, resp)
		if err != nil {
			fatal(err.Error())
		}

		fmt.Println(string(b))
	},
}

var membersCmd = &cobra.Command{
	Use:   "members",
	Short: "List members of a group",
	Run: func(cmd *cobra.Command, args []string) {
		cli := getClient(cmd)
		defer cli.Close()

		resp, err := search(cmd, cli)
		if err != nil {
			fatal(err.Error())
		}

		if len(resp.Entries) == 0 {
			fatal("group not found")
		}

		attributes, _ := cmd.Flags().GetStringSlice("attributes")
		if len(attributes) == 0 {
			attributes = []string{ldapcli.AttributeCommonName, ldapcli.AttributeObjectClass}
		}

		resp, err = cli.GroupMembersExtended(resp.Entries[0].DN, attributes...)
		if err != nil {
			fatal(err.Error())
		}

		output, _ := cmd.Flags().GetString("output")
		b, err := formatter.FormatLDAPSearchResult(output, resp)
		if err != nil {
			fatal(err.Error())
		}

		fmt.Println(string(b))
	},
}

var listCmd = &cobra.Command{
	Use:   "list <ou dn>",
	Short: "List members of an Organizational Unit",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cli := getClient(cmd)
		defer cli.Close()

		attributes, _ := cmd.Flags().GetStringSlice("attributes")
		if len(attributes) == 0 {
			attributes = []string{ldapcli.AttributeCommonName, ldapcli.AttributeObjectClass}
		}

		dn := cli.Config().BaseDN
		if len(args) > 0 {
			dn = args[0]
		}

		resp, err := cli.OrganizationalUnitMembers(dn, attributes...)
		if err != nil {
			fatal(err.Error())
		}

		output, _ := cmd.Flags().GetString("output")
		b, err := formatter.FormatLDAPSearchResult(output, resp)
		if err != nil {
			fatal(err.Error())
		}

		fmt.Println(string(b))
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func fatal(s string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, s+"\n", args...)
	os.Exit(1)
}

func getClient(cmd *cobra.Command) *ldapcli.Client {
	if err := viper.ReadInConfig(); err != nil {
		if !strings.Contains(err.Error(), "Not Found") {
			fatal(err.Error())
		}
		os.Mkdir(defaultConfigDir, 0700)
		viper.SetConfigFile(defaultConfigFile)
	}

	address := viper.GetString("address")
	if len(address) == 0 {
		fatal("no address configured")
	} else if !strings.HasPrefix(address, "ldap") {
		fatal("unkown address format: %s", address)
	}

	basedn := viper.GetString("basedn")
	if len(basedn) == 0 {
		basedn = ldapcli.ParseBaseDNFromDomain(address)
		viper.Set("basedn", basedn)
	}

	conf := ldapcli.NewConfig(address, basedn)
	conf.BindDN = viper.GetString("binddn")
	conf.BindPassword = viper.GetString("bindpw")
	conf.StartTLS = viper.GetBool("start-tls")
	conf.SkipVerify = viper.GetBool("insecure")

	if len(conf.BindDN) > 0 && len(conf.BindPassword) == 0 {
		fmt.Print("Enter password: ")
		bpw, _ := terminal.ReadPassword(int(syscall.Stdin))
		conf.BindPassword = strings.TrimSpace(string(bpw))
		viper.Set("bindpw", conf.BindPassword)
		fmt.Print("\n")
	}

	cli, err := ldapcli.Dial(conf)
	if err != nil {
		fatal(err.Error())
	}

	return cli
}

func search(cmd *cobra.Command, cli *ldapcli.Client) (*ldap.SearchResult, error) {
	attributes, _ := cmd.Flags().GetStringSlice("attributes")
	if len(attributes) == 0 {
		attributes = []string{ldapcli.AttributeCommonName, ldapcli.AttributeObjectClass}
	}

	dn, _ := cmd.Flags().GetString("dn")
	cn, _ := cmd.Flags().GetString("cn")
	byAttr, _ := cmd.Flags().GetString("by-attr")
	filter, _ := cmd.Flags().GetString("filter")

	if len(filter) == 0 {
		if len(dn) > 0 {
			if !ldapcli.IsDNSanitized(dn) {
				return nil, fmt.Errorf("dn contains invalid characters: %s", dn)
			}

			filter = fmt.Sprintf("(distinguishedName=%s)", dn)
		} else if len(cn) > 0 {
			if !ldapcli.IsNameSanitized(cn) {
				return nil, fmt.Errorf("cn contains invalid characters: %s", cn)
			}

			filter = fmt.Sprintf("(cn=%s)", cn)
		} else if len(byAttr) > 0 {
			if !strings.Contains(byAttr, "=") {
				return nil, fmt.Errorf("by-attr missing value to search for: %s", byAttr)
			}

			parts := strings.SplitN(byAttr, "=", 2)
			if !ldapcli.IsNameSanitized(parts[0]) {
				return nil, fmt.Errorf("by-attr name contains invalid characters: %s", parts[0])
			}
			if !ldapcli.IsDNSanitized(parts[1]) {
				return nil, fmt.Errorf("by-attr value contains invalid characters: %s", parts[1])
			}

			filter = fmt.Sprintf("(%s=%s)", parts[0], parts[1])
		}
	}

	if len(filter) == 0 {
		return nil, fmt.Errorf("search requires one of: --dn, --cn, --by-attr, --filter")
	}

	req := cli.NewSearchRequest(filter, attributes)
	return cli.Search(req)
}

func init() {
	rootCmd.PersistentFlags().StringP("address", "a", "", "Address to LDAP server in format: ldap://server.local:389 or ldaps://server.local:636")
	rootCmd.PersistentFlags().StringP("basedn", "b", "", "BaseDN for searching, defaults to auto discovery")
	rootCmd.PersistentFlags().StringP("binddn", "u", "", "BindDN to use for authentication")
	rootCmd.PersistentFlags().StringP("bindpw", "p", "", "Password to use for authentication, if not set you will be prompted")
	rootCmd.PersistentFlags().Bool("start-tls", false, "Start TLS")
	rootCmd.PersistentFlags().Bool("insecure", false, "Skip TLS validation errors")

	searchCmd.Flags().StringSlice("attributes", []string{}, "Comma-separated list of attributes to return")
	searchCmd.Flags().StringP("output", "o", "text", "Output format: json, json-pretty, ldif, text, yaml")
	searchCmd.Flags().String("dn", "", "Find by distingushedName")
	searchCmd.Flags().String("cn", "", "Find by common name (CN)")
	searchCmd.Flags().String("by-attr", "", "Find by attribute, format <attribute>=<value>")
	searchCmd.Flags().String("filter", "", "Find using LDAP filter")

	membersCmd.Flags().StringSlice("attributes", []string{}, "Comma-separated list of attributes to return")
	membersCmd.Flags().StringP("output", "o", "text", "Output format: json, json-pretty, ldif, text, yaml")
	membersCmd.Flags().String("dn", "", "Find by distingushedName")
	membersCmd.Flags().String("cn", "", "Find by common name (CN)")
	membersCmd.Flags().String("by-attr", "", "Find by attribute, format <attribute>=<value>")
	membersCmd.Flags().String("filter", "", "Find using LDAP filter")

	listCmd.Flags().StringSlice("attributes", []string{}, "Comma-separated list of attributes to return")
	listCmd.Flags().StringP("output", "o", "text", "Output format: json, json-pretty, ldif, text, yaml")

	rootCmd.AddCommand(loginCmd, searchCmd, membersCmd, listCmd)

	homeDir, _ := os.UserHomeDir()
	defaultConfigDir = homeDir + "/.direktor"
	defaultConfigFile = defaultConfigDir + "/direktorcli.yaml"

	viper.SetConfigName("direktorcli")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/direktor/")
	viper.AddConfigPath(".")
	viper.AddConfigPath(defaultConfigDir)
	viper.BindPFlag("address", rootCmd.PersistentFlags().Lookup("address"))
	viper.BindPFlag("basedn", rootCmd.PersistentFlags().Lookup("basedn"))
	viper.BindPFlag("binddn", rootCmd.PersistentFlags().Lookup("binddn"))
	viper.BindPFlag("bindpw", rootCmd.PersistentFlags().Lookup("bindpw"))
	viper.BindPFlag("start-tls", rootCmd.PersistentFlags().Lookup("start-tls"))
	viper.BindPFlag("insecure", rootCmd.PersistentFlags().Lookup("insecure"))
}
