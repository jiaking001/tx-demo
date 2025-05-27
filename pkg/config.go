package pkg

import (
	"errors"
	"github.com/spf13/viper"
)

func NewViper() *viper.Viper {
	v := viper.New()
	// 本地使用修改成config/local.yml
	v.SetConfigFile("config/local-test.yml")
	err := v.ReadInConfig()
	if err != nil {
		panic(errors.New("read config file failed: " + err.Error()))
	}
	return v
}
