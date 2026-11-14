// Package prompts loads versioned prompt templates from disk.
package prompts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Store struct {
	dir     string
	version string
	cache   map[string]string
}

func New(dir, version string) *Store {
	if version == "" {
		version = "v1"
	}
	return &Store{dir: dir, version: version, cache: map[string]string{}}
}

// Load reads prompts/<name>_<version>.txt (falls back to <name>.txt).
func (s *Store) Load(name string) (string, error) {
	if v, ok := s.cache[name]; ok {
		return v, nil
	}
	for _, fn := range []string{
		filepath.Join(s.dir, name+"_"+s.version+".txt"),
		filepath.Join(s.dir, name+".txt"),
	} {
		data, err := os.ReadFile(fn)
		if err == nil {
			text := strings.TrimSpace(string(data))
			s.cache[name] = text
			return text, nil
		}
	}
	return "", fmt.Errorf("prompt %q (version %s) not found in %s", name, s.version, s.dir)
}

// Must loads a prompt or panics — used at startup to fail fast on typos.
func (s *Store) Must(name string) string {
	p, err := s.Load(name)
	if err != nil {
		panic(err)
	}
	return p
}
