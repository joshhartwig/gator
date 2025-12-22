package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type Config struct {
	DB_URL            string `json:"db_url"`
	Current_User_Name string `json:"current_user_name"`
}

const configFileName = ".gatorconfig.json"

// Read reads the JSON file found in the gatorconfig.json file in the users HOME directory, decode the JSON into
// a Config struct and return the struct
func Read() (Config, error) {
	c := Config{}

	// get config path
	configPath, err := getConfigFilePath()
	if err != nil {
		return c, err
	}

	// read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return c, err
	}

	// decode file into struct
	err = json.NewDecoder(bytes.NewReader(data)).Decode(&c)
	if err != nil {
		return c, err
	}

	return c, nil
}

// SetUser sets the user on the Config struct to the user parameter
// it then passes the json data to write, to write the file to disk
func (c *Config) SetUser(user string) error {
	if user == "" {
		return errors.New("empty string entered")
	}

	// set the username
	c.Current_User_Name = user
	err := write(*c)
	if err != nil {
		return err
	}

	return nil
}

func getConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", homeDir, configFileName), nil
}

// write will write a Config struct to the file
func write(cfg Config) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	cfgFilePath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	err = os.WriteFile(cfgFilePath, data, os.ModeAppend)
	if err != nil {
		return err
	}

	return nil
}
