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

// Domain object represents an Active Directory domain connection.
type Domain struct {
	Name       string
	Address    string
	BindDN     string
	BindPW     string
	BaseDN     string
	StartTLS   bool
	SkipVerify bool
}

// Config object from environment configuration.
type Config struct {
	Domains []*Domain
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

	conf = &Config{}
	if err := viper.Unmarshal(conf); err != nil {
		return nil, err
	}

	return conf, nil
}

func init() {
	defaultConfigFile, _ := os.UserHomeDir()
	defaultConfigFile += "/.direktor.yaml"

	viper.SetConfigName("server")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/direktor/")
	viper.AddConfigPath("$HOME/.direktor")
	viper.AddConfigPath(".")
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		mu.Lock()
		conf = nil
		mu.Unlock()
	})
}
