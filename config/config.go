package config

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
)

type Config struct {
	Debug bool `yaml:"debug"`

	UriPrefix string `yaml:"uriprefix"`

	Server *ServerConf `yaml:"server"`

	Mongodb *MongodbConf `yaml:"mongodb"`
	Redis *RedisConf `yaml:"redis"`

	WeChat *WeChatLoginConf `yaml:"wechat"`

	PhoneSms *SmsConf `yaml:"phonesms"`
}

type ServerConf struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

func (s *ServerConf) GetServerAddr() string {
	return fmt.Sprintf("%s:%s", s.Host, s.Port)
}

type MongodbConf struct {
	Host string `yaml:"host" json:"host"`
	Port string `yaml:"port" json:"port"`
	User string `yaml:"user" json:"user"`
	Passwd string `yaml:"passwd" json:"passwd"`
	Database string `yaml:"database" json:"database"`
}

type RedisConf struct {
	Host string `yaml:"host" json:"host"`
	Port string `yaml:"port" json:"port"`
	Passwd string `yaml:"passwd" json:"passwd"`
	DbNum int `yaml:"dbnum" json:"dbnum"`
}

// 微信登录相关
type WeChatLoginConf struct {
	App *WeChatLoginAppConf `yaml:"app"`
	Web *WeChatLoginWebConf `yaml:"web"`
	XiaoChengXu *WeChatLoginAppConf `yaml:"xiaochengxu"`
}

// 微信 App 登录配置
type WeChatLoginAppConf struct {
	AppId string `yaml:"appid"`
	Secret string `yaml:"secret"`
}

// 微信 web 登录配置
type WeChatLoginWebConf struct {
	AppId string `yaml:"appid"`
	Secret string `yaml:"secret"`
	Token string `yaml:"token"`
	AesKey string `yaml:"aeskey"`
}

// 短信
type SmsConf struct {
	Account string `yaml:"account"`
	Password string `yaml:"password"`
	Url string `yaml:"url"`
	Debug bool `yaml:"debug"`
}

func LoadConf(path string) (*Config, error) {
	if path == "" {
		return nil, errors.New("path不能为空")
	}

	viper.SetConfigFile(path)
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("读取配置文件失败", err.Error())
		return nil, err
	}

	var c Config
	err = viper.Unmarshal(&c)
	if err != nil {
		fmt.Println("读取配置文件失败", err.Error())
		return nil, err
	}

	return &c, nil
}