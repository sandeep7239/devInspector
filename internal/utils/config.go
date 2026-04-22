package utils

import (
	"errors"
	"os"

	"github.com/sandeep7239/devInspector/pkg/models"
	"gopkg.in/yaml.v3"
)

const ConfigFileName = ".devinspector.yaml"

func DefaultConfig() models.Config {
	return models.Config{
		WorkerCount:    5,
		FailOnCritical: true,
	}
}

func LoadConfig(projectPath string) (models.Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(projectPath + string(os.PathSeparator) + ConfigFileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 5
	}
	return cfg, nil
}

func WriteDefaultConfig(projectPath string) error {
	cfg := DefaultConfig()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(projectPath+string(os.PathSeparator)+ConfigFileName, data, 0644)
}
