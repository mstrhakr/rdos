package main

import (
	"bufio"
	"os"
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
			var t int
			if _, err := sscanf(val, "%d", &t); err == nil && t > 0 {
				cfg.PinTimeout = t
			}
		}
	}

	return cfg, scanner.Err()
}

// Simple sscanf-like function for basic parsing
func sscanf(s string, format string, args ...interface{}) (int, error) {
	if format == "%d" && len(args) > 0 {
		if pi, ok := args[0].(*int); ok {
			var num int
			_, err := parseInteger(s, &num)
			if err == nil {
				*pi = num
				return 1, nil
			}
			return 0, err
		}
	}
	return 0, nil
}

func parseInteger(s string, pi *int) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	var result int
	negative := false
	if s[0] == '-' {
		negative = true
		s = s[1:]
	}

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch < '0' || ch > '9' {
			break
		}
		result = result*10 + int(ch-'0')
	}

	if negative {
		result = -result
	}

	*pi = result
	return 1, nil
}
