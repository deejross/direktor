package config

import (
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var (
	mu   = sync.RWMutex{}
	conf *Config
)

// Domain object represents an Active Directory domain connection. Because we make it easy for users to log in using their own credentials,
// we don't require users to enter/know their full DN. Instead, we find the DN using a search comparing their username against the `UsernameAttribute`.
// On LDAP systems that don't allow anonymous searches, an `InitialBindDN` and `InitialBindPW` is required for the search. This can be a read-only account
// as it is not used for anything other than searching for user DNs during the login process. All other operations against the directory are performed
// as the logged in user.
type Domain struct {
	Name              string // Friendly name of the domain
	Address           string // URI Address if the server with scheme (i.e. ldap://server.local:389)
	UsernameAttribute string // Attribute to use for username when searching for user DN during binding
	InitialBindDN     string // Initial bind DN to use for searching for given username's DN, required if anonymous binding is disabled and should be a limited read-only account, defaults to `cn`
	InitialBindPW     string // The password for the InitialBindDN, may be required if InitialBindDN is set
	BaseDN            string // Default base DN used for searching
	StartTLS          bool   // Should this connection StartTLS, only used for non-LDAPS connections
	SkipVerify        bool   // Skip server TLS certificate verification
}

// Config object from environment configuration.
type Config struct {
	Domains    []*Domain // List of domains to configure
	ListenPort string    // The port number for the server to listen on, defaults to 8000 or value of PORT environment variable
}

// Get reads in the configuration from the config file and returns a Config object.
// The result is cached for future calls, and cache is invalidated automatically if
// the config file is modified.
func Get() (*Config, error) {
	mu.RLock()
	if conf != nil {
		mu.RUnlock()
		return conf, nil
	}

	mu.RUnlock()
	mu.Lock()
	defer mu.Unlock()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	conf = &Config{}
	if err := viper.Unmarshal(conf); err != nil {
		return nil, err
	}

	// determine the ListenPort if not set in config file
	if len(conf.ListenPort) == 0 {
		if port := os.Getenv("PORT"); len(port) > 0 {
			conf.ListenPort = port
		} else {
			conf.ListenPort = "8000"
		}
	}

	return conf, nil
}

func init() {
	homeDir, _ := os.UserHomeDir()
	defaultConfigDir := homeDir + "/.direktor"

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/direktor/")
	viper.AddConfigPath(".")
	viper.AddConfigPath(defaultConfigDir)
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		mu.Lock()
		conf = nil
		mu.Unlock()
	})
}
