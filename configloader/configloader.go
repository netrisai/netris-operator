package configloader

import (
	"errors"
	"log"
	"os"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
)

const ymlExt = ".yml"

func readFile(path string, configPtr interface{}) error {
	f, openErr := os.Open(path)
	if openErr != nil {
		return openErr
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	decErr := decoder.Decode(configPtr)
	if decErr != nil {
		return decErr
	}

	return nil
}

func readEnv(configPtr interface{}) error {
	err := envconfig.Process("", configPtr)
	if err != nil {
		return err
	}

	return nil
}

// Unmarshal "path" yml file path
// "configPtr" pointer to result struct
func Unmarshal(path string, configPtr interface{}) error {
	extension := path[len(path)-4:]
	if extension != ymlExt {
		return errors.New("only " + ymlExt + " file supported")
	}

	fileErr := readFile(path, configPtr)
	if fileErr != nil {
		log.Println("config.yml not loaded")
	}

	envErr := readEnv(configPtr)
	if envErr != nil {
		return envErr
	}

	return nil
}
