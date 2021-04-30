package config

import (
	"strings"

	"github.com/spf13/viper"
)

func Load(file string) error {
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/schiff-operator")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.WatchConfig()

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if file != "" {
		viper.SetConfigFile(file)
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}
	return nil
}
