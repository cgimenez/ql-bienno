package main

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

type InstanceConfig struct {
	Image        string `validate:"required"`
	IpAddress    string `yaml:"ip_address" validate:"required"`
	Gateway      string
	SshPubKey    string `yaml:"ssh_pub_key" validate:"required"`
	Smp          int    `validate:"gt=0"`
	Mem          int    `validate:"gt=0"`
	DiskSize     int    `yaml:"disk_size" validate:"gt=0"`
	UserName     string `yaml:"user_name" validate:"required"`
	Samba        bool
	HostUser     string `yaml:"host_user"`
	EnableVirtFS bool   `yaml:"enable_virtfs"`
}

func buildInstanceConfig() *InstanceConfig {
	return &InstanceConfig{
		Smp:          2,
		Mem:          8,
		DiskSize:     40,
		Gateway:      "192.168.1.254",
		Samba:        false,
		EnableVirtFS: false,
	}
}

func (conf *InstanceConfig) Load(config_file string) error {
	var err error
	var data []byte
	data, err = os.ReadFile(config_file)
	if err != nil {
		return fmt.Errorf("Reading config file %s : %s", config_file, err)
	}

	err = yaml.Unmarshal([]byte(data), &conf)
	if err != nil {
		return fmt.Errorf("Parsing yaml file %s : %s", config_file, err)
	}

	var validate = validator.New(validator.WithRequiredStructEnabled())
	err = validate.Struct(conf)
	if err != nil {
		return fmt.Errorf("Validating config file %s : %s", config_file, err)
	}

	return nil
}
