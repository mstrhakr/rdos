package tcconfig

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var keyPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func ValidKey(key string) bool {
	return keyPattern.MatchString(strings.TrimSpace(key))
}

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Load() (map[string]string, error) {
	cfg := map[string]string{}

	f, err := os.Open(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, fmt.Errorf("open tcconfig: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		idx := strings.IndexRune(line, '=')
		if idx <= 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		if !keyPattern.MatchString(key) {
			continue
		}

		value := strings.TrimSpace(line[idx+1:])
		cfg[key] = unquote(value)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan tcconfig: %w", err)
	}

	return cfg, nil
}

func (s *Store) Save(cfg map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create tcconfig parent dir: %w", err)
	}

	tmp := s.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open temp tcconfig: %w", err)
	}

	keys := make([]string, 0, len(cfg))
	for key := range cfg {
		if ValidKey(key) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	w := bufio.NewWriter(f)
	for _, key := range keys {
		if _, err := fmt.Fprintf(w, "%s=%s\n", key, quote(cfg[key])); err != nil {
			_ = f.Close()
			_ = os.Remove(tmp)
			return fmt.Errorf("write tcconfig: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("flush tcconfig: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close tcconfig: %w", err)
	}

	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace tcconfig: %w", err)
	}

	return nil
}

func unquote(v string) string {
	if len(v) >= 2 && strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"") {
		v = v[1 : len(v)-1]
	}
	v = strings.ReplaceAll(v, `\\`, `\`)
	v = strings.ReplaceAll(v, `\"`, `"`)
	return v
}

func quote(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `"`, `\"`)
	return `"` + v + `"`
}
