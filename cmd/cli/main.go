package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/deejross/direktor/pkg/formatter"
	"github.com/deejross/direktor/pkg/ldapcli"
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

var searchCmd = &cobra.Command{
	Use:   "search <filter>",
	Short: "Search using LDAP filter",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		conf := getConfig(cmd)
		cli, err := ldapcli.Dial(conf)
		if err != nil {
			fatal(err.Error())
		}

		attributes, _ := cmd.Flags().GetStringSlice("attributes")
		if len(attributes) == 0 {
			attributes = []string{ldapcli.AttributeCommonName, ldapcli.AttributeObjectClass}
		}

		output, _ := cmd.Flags().GetString("output")
		req := cli.NewSearchRequest(args[0], attributes)
		resp, err := cli.Search(req)
		if err != nil {
			fatal(err.Error())
		}

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

func getConfig(cmd *cobra.Command) *ldapcli.Config {
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

	if err := viper.WriteConfig(); err != nil {
		os.Stderr.WriteString(fmt.Sprintf("WARN: unable to write state: %s\n", err))
	}

	return conf
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

	rootCmd.AddCommand(searchCmd)

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
