package config

import (
	"github.com/ansel1/merry"
	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Kubernetes KubernetesConfig `mapstructure:"kubernetes"`
	Log        LogConfig        `mapstructure:"log"`
	Envoy      EnvoyConfig      `mapstructure:"envoy"`
}

type ServerConfig struct {
	Address string `mapstructure:"address"`
}

type KubernetesConfig struct {
	Namespace string `mapstructure:"namespace"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type EnvoyConfig struct {
	Node string `mapstructure:"node"`
}

func ReadConfig() (*Config, error) {
	var config Config
	v := viper.New()

	// Bind to environment variables
	v.AutomaticEnv()

	// Default values
	v.SetDefault("server.address", ":4000")
	v.SetDefault("kubernetes.namespace", "default")
	v.SetDefault("log.level", "info")

	if err := v.Unmarshal(&config); err != nil {
		return nil, merry.Wrap(err)
	}

	return &config, nil
}

func MustReadConfig() *Config {
	conf, err := ReadConfig()

	if err != nil {
		panic(err)
	}

	return conf
}
