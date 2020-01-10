package config

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
)

// Config holder for parsed config
type Config struct {
	Exec    string            `yaml:"exec"`
	Args    map[string]string `yaml:"args"`
	SRV     []string          ` yaml:"srv"`
	JoinMax int               `yaml:"join-max"`
}

// ExecCmd generated the start command
func (c *Config) ExecCmd() string {
	var cmdBuilder strings.Builder
	cmdBuilder.WriteString(fmt.Sprintf("%s start ", c.Exec))
	for k, v := range c.Args {
		cmdBuilder.WriteString(fmt.Sprintf("--%s=\"%s\" ", k, v))
	}
	return cmdBuilder.String()
}

// SetLocality sets locality that overrides what was read from yaml config file
func (c *Config) SetLocality(locality string) {
	c.Args["locality"] = locality
}

// SetJoin sets join list that overrides what was read from yaml config file
func (c *Config) SetJoin(join string) {
	c.Args["join"] = join
}

// Read unmarshals config yaml
func Read(yamlPath string) (*Config, error) {
	confBytes, err := ioutil.ReadFile(yamlPath)
	if err != nil {
		return nil, err
	}

	//m := make(map[string]string)
	c := &Config{}

	err = yaml.Unmarshal(confBytes, &c)
	if err != nil {
		return nil, err
	}
	return c, nil
}
