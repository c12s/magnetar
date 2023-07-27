package configs

import (
	"os"
)

type Config struct {
	natsAddress         string
	etcdAddress         string
	registrationSubject string
}

func (c *Config) NatsAddress() string {
	return c.natsAddress
}

func (c *Config) EtcdAddress() string {
	return c.etcdAddress
}

func (c *Config) RegistrationSubject() string {
	return c.registrationSubject
}

func NewFromEnv() (*Config, error) {
	return &Config{
		natsAddress:         os.Getenv("NATS_ADDRESS"),
		etcdAddress:         os.Getenv("ETCD_ADDRESS"),
		registrationSubject: os.Getenv("REGISTRATION_SUBJECT"),
	}, nil
}