package config

import "path/filepath"

type DataBaseConfig struct {
	Name    string // Database Name
	DataDir string // Database Path
}

func (c *DataBaseConfig) ResolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if c.DataDir == "" {
		return ""
	}
	return filepath.Join(c.instanceDir(), path)
}

func (c *DataBaseConfig) instanceDir() string {
	if c.DataDir == "" {
		return ""
	}
	return filepath.Join(c.DataDir, c.Name)
}
