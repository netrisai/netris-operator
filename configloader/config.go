package configloader

import (
	"errors"
	"log"
	"os"
	"path"
)

type config struct {
	Controller controller `yaml:"controller"`
}

type controller struct {
	Host     string `yaml:"address" envconfig:"CONTROLLER_HOST"`
	Login    string `yaml:"login" envconfig:"CONTROLLER_LOGIN"`
	Password string `yaml:"password" envconfig:"CONTROLLER_PASSWORD"`
}

var Root *config

func init() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	ptr := &config{}
	err = Unmarshal(path.Join(wd, "configloader", "config.yml"), ptr)
	Root = ptr
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("%s/configloader/config.yml not found. Config not loaded, no panic for tests", wd)
		} else {
			panic(err)
		}
	} else {
		log.Printf("%s/configloader/config.yml config loaded", wd)
	}
}
