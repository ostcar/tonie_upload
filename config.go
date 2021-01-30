package main

import (
	"fmt"
	"os"

	"github.com/gen2brain/dlgs"
	"gopkg.in/yaml.v3"
)

const configFile = "tonie_upload.yml"

type config struct {
	Username    string
	Password    string
	HouseholdID string `yaml:"household_id"`
	TonieID     string `yaml:"tonie_id"`
}

func loadConfig() (config, error) {
	path, err := os.UserConfigDir()
	if err != nil {
		return config{}, fmt.Errorf("getting config path: %w", err)
	}

	f, err := os.Open(path + "/" + configFile)
	if err != nil {
		return config{}, err
	}

	var conf config
	if err := yaml.NewDecoder(f).Decode(&conf); err != nil {
		return config{}, fmt.Errorf("decoding config: %w", err)
	}

	return conf, nil
}

func writeConfig(conf config) error {
	path, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("getting config path: %w", err)
	}

	f, err := os.Create(path + "/" + configFile)
	if err != nil {
		return fmt.Errorf("creating config file: %w", err)
	}

	if err := yaml.NewEncoder(f).Encode(conf); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	return nil

}

func configWizzard() (config, error) {
	username, ok, err := dlgs.Entry("Tonie upload", "Tonie Username:", "")
	if err != nil {
		return config{}, fmt.Errorf("getting username: %v", err)
	}
	if !ok {
		return config{}, fmt.Errorf("abort")
	}

	password, ok, err := dlgs.Password("Tonie upload", "Tonie Password:")
	if err != nil {
		return config{}, fmt.Errorf("getting password: %v", err)
	}
	if !ok {
		return config{}, fmt.Errorf("abort")
	}

	conf := config{Username: username, Password: password}

	c, err := newConnection(conf)
	if err != nil {
		return config{}, fmt.Errorf("creating connection: %w", err)
	}

	// Households
	households, err := c.households()
	if err != nil {
		return config{}, fmt.Errorf("getting households: %w", err)
	}

	var householdItems []string
	for name := range households {
		householdItems = append(householdItems, name)
	}

	household, ok, err := dlgs.List("Tonie upload", "choose household", householdItems)
	if err != nil {
		return config{}, fmt.Errorf("getting household: %v", err)
	}
	if !ok {
		return config{}, fmt.Errorf("abort")
	}
	conf.HouseholdID = households[household]

	// Tonies
	tonies, err := c.tonies(conf.HouseholdID)
	if err != nil {
		return config{}, fmt.Errorf("getting tonies: %w", err)
	}

	var tonieItems []string
	for name := range tonies {
		tonieItems = append(tonieItems, name)
	}

	tonie, ok, err := dlgs.List("Tonie upload", "choose household", tonieItems)
	if err != nil {
		return config{}, fmt.Errorf("getting tonies: %v", err)
	}
	if !ok {
		return config{}, fmt.Errorf("abort")
	}
	conf.TonieID = tonies[tonie]

	// Write config
	if err := writeConfig(conf); err != nil {
		return config{}, fmt.Errorf("saving config: %w", err)
	}
	return conf, nil
}
