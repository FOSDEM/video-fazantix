package config

import (
	"path/filepath"

	yaml "github.com/goccy/go-yaml"
)

type CfgPath string

// UnmarshalBase is a hack that must be thrown into the sun
var UnmarshalBase string

func (c *CfgPath) UnmarshalYAML(b []byte) error {
	var path string

	err := yaml.Unmarshal(b, &path)
	if err != nil {
		return err
	}

	if filepath.IsAbs(path) {
		*c = CfgPath(path)
	} else {
		*c = CfgPath(filepath.Join(UnmarshalBase, path))
	}
	return nil
}
