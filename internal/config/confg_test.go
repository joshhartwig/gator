package config

import (
	"testing"
)

func TestGetConfigFile(t *testing.T) {
	want := "/Users/joshuahartwig/.gatorconfig.json"
	got, err := getConfigFilePath()
	if err != nil {
		t.Errorf("unexpected error %s", err.Error())
	}

	if got != want {
		t.Errorf("wanted %s got %s", want, got)
	}
}

func TestSetUser(t *testing.T) {
	cfg := Config{}
	cfg.Current_User_Name = "tom"

	cfg.SetUser("bob")

	if cfg.Current_User_Name != "bob" {
		t.Errorf("error setting user name")
	}
}
