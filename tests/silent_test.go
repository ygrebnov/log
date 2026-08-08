package tests

import (
	"testing"

	keyslib "github.com/ygrebnov/keys"
	"github.com/ygrebnov/log"
	"github.com/ygrebnov/log/pkg/config"
)

func TestLogger_Silent(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
	}{
		{
			name: "nil config",
			cfg:  nil,
		},
		{
			name: "empty config",
			cfg:  &config.Config{},
		},
		{
			name: "empty sinks",
			cfg: &config.Config{
				Sinks: []config.SinkConfig{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := log.NewLogger(tt.cfg)
			if err != nil {
				t.Fatalf("NewLogger() error: %v", err)
			}

			logger.Log(
				log.LevelInfo,
				"must be silently discarded",
				log.String(keyslib.New("log.message"), "value"),
			)

			if err := logger.Close(); err != nil {
				t.Fatalf("Close() error: %v", err)
			}
		})
	}
}

func TestNewSilentLogger(t *testing.T) {
	logger, err := log.NewSilentLogger()
	if err != nil {
		t.Fatalf("NewSilentLogger() error: %v", err)
	}

	logger.Log(
		log.LevelInfo,
		"must be silently discarded",
		log.String(keyslib.New("log.message"), "value"),
	)

	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}
