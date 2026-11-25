// Package config loads and validates Mavis configuration.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const (
	// FileName is the configuration filename in the served content root.
	FileName = ".mavis.conf"
	// DefaultHost is the loopback-only default TCP host.
	DefaultHost = "127.0.0.1"
	// DefaultPort is the default TCP port.
	DefaultPort = 3344
)

// Server contains listener settings.
type Server struct {
	Host   string
	Port   int
	Socket string
}

// Config contains the effective settings for one serving session.
type Config struct {
	Exclude       []string
	IncludeHidden bool
	Server        Server

	hostSet   bool
	portSet   bool
	socketSet bool
}

// Overrides contains explicitly supplied CLI values. Nil fields are unchanged.
type Overrides struct {
	Host          *string
	Port          *int
	Socket        *string
	IncludeHidden *bool
	Exclude       []string
}

type fileConfig struct {
	Exclude       []string   `toml:"exclude"`
	IncludeHidden *bool      `toml:"include_hidden"`
	Server        fileServer `toml:"server"`
}

type fileServer struct {
	Host   *string `toml:"host"`
	Port   *int    `toml:"port"`
	Socket *string `toml:"socket"`
}

// Default returns the safe serving defaults.
func Default() Config {
	return Config{
		Server: Server{
			Host: DefaultHost,
			Port: DefaultPort,
		},
	}
}

// Load reads optional configuration from the served root.
func Load(root string) (Config, error) {
	configuration := Default()
	data, err := os.ReadFile(filepath.Join(root, FileName))
	if errors.Is(err, fs.ErrNotExist) {
		return configuration, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read %s: %w", FileName, err)
	}

	var file fileConfig
	if err := decode(data, &file); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", FileName, err)
	}

	configuration.Exclude = append([]string(nil), file.Exclude...)
	if file.IncludeHidden != nil {
		configuration.IncludeHidden = *file.IncludeHidden
	}
	if file.Server.Host != nil {
		configuration.Server.Host = *file.Server.Host
		configuration.hostSet = true
	}
	if file.Server.Port != nil {
		configuration.Server.Port = *file.Server.Port
		configuration.portSet = true
	}
	if file.Server.Socket != nil {
		configuration.Server.Socket = *file.Server.Socket
		configuration.socketSet = true
	}

	if err := configuration.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid %s: %w", FileName, err)
	}
	return configuration, nil
}

// Merge applies explicit CLI values and combines both exclusion sources.
func Merge(configuration Config, overrides Overrides) (Config, error) {
	merged := configuration
	merged.Exclude = append([]string(nil), configuration.Exclude...)
	if overrides.Host != nil {
		merged.Server.Host = *overrides.Host
		merged.hostSet = true
	}
	if overrides.Port != nil {
		merged.Server.Port = *overrides.Port
		merged.portSet = true
	}
	if overrides.Socket != nil {
		merged.Server.Socket = *overrides.Socket
		merged.socketSet = true
	}
	if overrides.IncludeHidden != nil {
		merged.IncludeHidden = *overrides.IncludeHidden
	}
	merged.Exclude = append(merged.Exclude, overrides.Exclude...)

	if err := merged.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid serving configuration: %w", err)
	}
	return merged, nil
}

// Validate checks effective values and explicit listener selection.
func (configuration Config) Validate() error {
	if err := configuration.validateValues(); err != nil {
		return err
	}
	if configuration.socketSet && (configuration.hostSet || configuration.portSet) {
		settings := []string{"server.socket"}
		if configuration.hostSet {
			settings = append(settings, "server.host")
		}
		if configuration.portSet {
			settings = append(settings, "server.port")
		}
		return fmt.Errorf("%s select conflicting listeners; choose TCP or a Unix socket", strings.Join(settings, " and "))
	}
	if _, err := CompileExclusions(configuration.Exclude); err != nil {
		return err
	}
	return nil
}

func decode(data []byte, target any) error {
	decoder := toml.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		var missing *toml.StrictMissingError
		if errors.As(err, &missing) && len(missing.Errors) > 0 {
			key := strings.Join(missing.Errors[0].Key(), ".")
			return fmt.Errorf("unknown setting %q: %w", key, err)
		}
		var decodeError *toml.DecodeError
		if errors.As(err, &decodeError) && len(decodeError.Key()) > 0 {
			key := strings.Join(decodeError.Key(), ".")
			return fmt.Errorf("invalid setting %q: %w", key, err)
		}
		return err
	}
	return nil
}

func (configuration Config) validateValues() error {
	if strings.TrimSpace(configuration.Server.Host) == "" {
		return errors.New("server.host must not be empty")
	}
	if configuration.Server.Port < 1 || configuration.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", configuration.Server.Port)
	}
	if configuration.socketSet && configuration.Server.Socket == "" {
		return errors.New("server.socket must not be empty")
	}
	return nil
}
