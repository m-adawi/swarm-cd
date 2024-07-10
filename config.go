package main

import (
	"github.com/spf13/viper"
)


func initConfigs() (err error) {
	err = readRepoConfigs()
	if err != nil {
		return 
	}
	err = readStackConfigs()
	if err != nil {
		return 
	}
	return
}


func readRepoConfigs() (err error) {
	reposViper := viper.New()
	reposViper.SetConfigName("repos")
	reposViper.AddConfigPath(".")
	err = reposViper.ReadInConfig()
	if err != nil {
		return
	}
	err = reposViper.Unmarshal(&repoConfigs)
	if err != nil {
		return
	}
	return
}


func readStackConfigs() (err error) {
	stacksViper := viper.New()
	stacksViper.SetConfigName("stacks")
	stacksViper.AddConfigPath(".")
	err = stacksViper.ReadInConfig()
	if err != nil {
		return
	}
	err = stacksViper.UnmarshalKey("stacks", &stackConfigs)
	if err != nil {
		return
	}
	return
}
