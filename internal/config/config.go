package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var (
	mu   = sync.RWMutex{}
	conf *Config
)

// Config object from environment configuration.
type Config struct {
	ListenPort string // The port number for the server to listen on, defaults to 8000 or value of PORT environment variable
	SecretKey  string // The secret key used for signing and encrypting authentication tokens
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

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	// determine the ListenPort if not set in config file
	if len(config.ListenPort) == 0 {
		if port := os.Getenv("PORT"); len(port) > 0 {
			config.ListenPort = port
		} else {
			config.ListenPort = "8000"
		}
	}

	// SecretKey is required
	if len(config.SecretKey) == 0 {
		return nil, fmt.Errorf("secretKey is a required field")
	}

	Set(config)
	return config, nil
}

// Set the current configuration. Will be overriden if config file changes.
// This is mostly used for unit/integration testing.
func Set(config *Config) {
	mu.Lock()
	conf = config
	mu.Unlock()
}

func init() {
	homeDir, _ := os.UserHomeDir()
	defaultConfigDir := homeDir + "/.direktor"

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/direktor/")
	viper.AddConfigPath(".")
	viper.AddConfigPath(defaultConfigDir)
	viper.AutomaticEnv()
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		mu.Lock()
		conf = nil
		mu.Unlock()
	})
}
