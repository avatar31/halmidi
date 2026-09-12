package config

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/avatar31/dotfs/fileutils"
	"go.uber.org/zap/zapcore"
)

const (
	APP_NAME = "halmidi"

	CONFIG_FILE_PATH = "/etc/halmidi/halmidi.conf"
	DEFAULT_LOG_PATH = "/var/log/halmidi"
)

var config *Config

type Logging struct {
	Path  string
	Level zapcore.Level
}

type Config struct {
	Logging Logging
}

func (c *Config) loadConfig(configMap map[string]map[string]string) error {
	// mode, ok := configMap["general"]["mode"]
	// if ok {
	// 	mode = strings.ToUpper(mode)
	// 	if mode != MODE_STANDALONE && mode != MODE_CLUSTER {
	// 		mode = MODE_STANDALONE
	// 	}
	// } else {
	// 	mode = MODE_STANDALONE
	// }
	// c.Mode = mode

	// err := c.setMetadataConfig(configMap)
	// if err != nil {
	// 	return err
	// }

	// err = c.setVolumesConfig(configMap)
	// if err != nil {
	// 	return err
	// }

	c.setLogConfig(configMap)
	return nil
}

func (c *Config) setLogConfig(configMap map[string]map[string]string) {
	logpath := DEFAULT_LOG_PATH
	logLevel := zapcore.InfoLevel
	loggingConf, ok := configMap["logging"]
	if ok {
		logpath, ok = loggingConf["path"]
		if !ok {
			logpath = DEFAULT_LOG_PATH
		}

		if level, ok := loggingConf["log-level"]; ok {
			switch strings.ToLower(level) {
			case "debug":
				logLevel = zapcore.DebugLevel
			case "error":
				logLevel = zapcore.ErrorLevel
			default:
				logLevel = zapcore.InfoLevel
			}
		}
	}

	c.Logging = Logging{
		Path:  logpath,
		Level: logLevel,
	}
}

func GetConfig() *Config {
	if config != nil {
		return config
	}

	_, err := LoadConfig(CONFIG_FILE_PATH)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	return config
}

func LoadConfig(configFile string) (*Config, error) {
	if !fileutils.IsFileExists(configFile) {
		return nil, fmt.Errorf("error opening config file %s", configFile)
	}

	configMap, err := readConfFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %v", err)
	}

	config = &Config{}
	err = config.loadConfig(configMap)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func readConfFile(configFile string) (map[string]map[string]string, error) {
	conf := make(map[string]map[string]string)

	file, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	currentSection := "general"
	conf[currentSection] = make(map[string]string) // [general] is for general purpose app settings

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments starting with # or ;
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Check for section header
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.TrimSpace(line[1 : len(line)-1])
			if _, exists := conf[currentSection]; !exists {
				conf[currentSection] = make(map[string]string)
			}
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // ignore malformed lines
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		conf[currentSection][key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if _, ok := conf["general"]["basepath"]; !ok {
		return nil, fmt.Errorf("basepath is not defined in [general] section of config file")
	}

	return conf, nil
}
