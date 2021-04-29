package infoblox

import (
	log "github.com/sirupsen/logrus"
	. "github.com/spf13/viper"
)

// LoadConfig loads necessary configuration to connect to Infoblox client from a configuration file
func LoadConfig(path, configName, configType string) (config InfobloxConfig, err error) {
	AddConfigPath(path)
	SetConfigName(configName)
	SetConfigType(configType)
	AutomaticEnv()

	err = ReadInConfig()
	if err != nil {
		log.Error(err, "Could not read from config file")
		return
	}
	err = Unmarshal(&config)
	if err != nil {
		log.Error("Could not unmarshal config")
	}
	return
}
