package infoblox

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// LoadConfig loads necessary configuration to connect to Infoblox client from a configuration file
func LoadConfig(path, configName, configType string) (config InfobloxConfig, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName(configName)
	viper.SetConfigType(configType)
	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		log.Error(err, "Could not read from config file")
	}
	err = viper.Unmarshal(&config)
	return
}
