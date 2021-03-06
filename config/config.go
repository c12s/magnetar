package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Celestial struct {
	Conf Config `yaml:"magnetar"`
}

type Config struct {
	ConfVersion    string            `yaml:"version"`
	Address        string            `yaml:"address"`
	Apollo         string            `yaml:"apollo"`
	Meridian       string            `yaml:"meridian"`
	Endpoints      []string          `yaml:"db"`
	Metrics        string            `yaml:"mdb"`
	Sync           string            `yaml:"sync"`
	Health         string            `yaml:"health"`
	InstrumentConf map[string]string `yaml:"instrument"`
}

func ConfigFile(n ...string) (*Config, error) {
	path := "config.yml"
	if len(n) > 0 {
		path = n[0]
	}

	yamlFile, err := ioutil.ReadFile(path)
	check(err)

	var conf Celestial
	err = yaml.Unmarshal(yamlFile, &conf)
	check(err)

	return &conf.Conf, nil
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
