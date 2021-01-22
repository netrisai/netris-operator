package configloader

import (
	"log"
	"os"
	"path"
)

type config struct {
	Controller controller `yaml:"controller"`
}

type controller struct {
	Host     string `yaml:"host" envconfig:"CONTROLLER_HOST"`
	Login    string `yaml:"login" envconfig:"CONTROLLER_LOGIN"`
	Password string `yaml:"password" envconfig:"CONTROLLER_PASSWORD"`
	Insecure bool   `yaml:"insecure" envconfig:"CONTROLLER_INSECURE"`
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
		log.Fatalf("configloader error: %v", err)
	} else {
		if len(ptr.Controller.Host) == 0 {
			log.Fatalln("Please set netris controller credentials")
		} else {
			log.Printf("connecting to host - %v", ptr.Controller.Host)
		}
	}
}
