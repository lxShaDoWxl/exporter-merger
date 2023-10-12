package cmd

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	prom "github.com/prometheus/client_model/go"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Exporters []Exporter
}

type Exporter struct {
	URL       string            `yaml:"url"`
	AddLabels []*prom.LabelPair `yaml:"addLabels"`
}

func ReadConfig(path string) (*Config, error) {
	var err error

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	config := new(Config)
	err = yaml.Unmarshal(raw, config)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse %s", path)
	}

	log.WithFields(log.Fields{
		"content": fmt.Sprintf("%#v", config),
		"path":    path,
	}).Debug("loaded config file")

	return config, nil
}
