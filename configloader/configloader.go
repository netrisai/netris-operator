package configloader

import (
	"errors"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
	"os"
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

// "path" yml file path
// "configPtr" pointer to result struct
func Unmarshal(path string, configPtr interface{}) error {
	extension := path[len(path)-4:]
	if extension != ymlExt {
		return errors.New("only " + ymlExt + " file supported")
	}

	fileErr := readFile(path, configPtr)
	if fileErr != nil {
		return fileErr
	}

	envErr := readEnv(configPtr)
	if envErr != nil {
		return envErr
	}

	return nil
}
