package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Theme      string
	Enabled    bool
	Position   string
	PinTimeout int
}

func (c *Config) SetDefaults() {
	c.Theme = "dark"
	c.Enabled = true
	c.Position = "top"
	c.PinTimeout = 5
}

func LoadConfig(path string) (*Config, error) {
	cfg := &Config{}
	cfg.SetDefaults()

	f, err := os.Open(path)
	if err != nil {
		return cfg, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "overlay_theme":
			if val == "light" || val == "dark" {
				cfg.Theme = val
			}
		case "overlay_enabled":
			cfg.Enabled = val == "true" || val == "1" || val == "yes"
		case "overlay_position":
			if val == "top" || val == "bottom" {
				cfg.Position = val
			}
		case "overlay_pin_timeout":
			if t, err := strconv.Atoi(val); err == nil && t > 0 {
				cfg.PinTimeout = t
			}
		}
	}

	return cfg, scanner.Err()
}
