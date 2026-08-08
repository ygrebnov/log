package fs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ygrebnov/errorc"
	"github.com/ygrebnov/log/pkg/config"
	errorspkg "github.com/ygrebnov/log/pkg/errors"
	"github.com/ygrebnov/log/pkg/keys"
)

type Closer interface {
	Close() error
}

type nopCloser struct{}

func (nopCloser) Close() error {
	return nil
}

// OpenWriter opens a writer for the configured log destination.
func OpenWriter(cfg *config.SinkConfig, appName string) (w io.Writer, c Closer, e error) {
	var doNotCloseCloser bool

	defer func() {
		if !doNotCloseCloser && e != nil && c != nil {
			e = errors.Join(e, c.Close())
		}
	}()

	path := strings.TrimSpace(cfg.Path)
	if path != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			doNotCloseCloser = true

			return nil, nopCloser{}, errorc.With(
				errorspkg.ErrCannotAccessLogFilePath,
				errorc.String(keys.LogFilePath, path),
				errorc.Error(keys.Cause, err),
			)
		}

		return openFile(path)
	}

	logDir, err := getLogDir(appName)
	if err != nil {
		return nil, nopCloser{}, err
	}

	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return os.Stderr, nopCloser{}, errorc.With(
			errorspkg.ErrCannotAccessLogFilePath,
			errorc.String(keys.LogDir, logDir),
			errorc.Error(keys.Cause, err),
		)
	}

	return openFile(filepath.Join(logDir, appName+".log"))
}

func openFile(path string) (io.Writer, Closer, error) {
	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0o600,
	)
	if err != nil {
		return os.Stderr, nopCloser{}, errorc.With(
			errorspkg.ErrCannotAccessLogFilePath,
			errorc.String(keys.LogFilePath, path),
			errorc.Error(keys.Cause, err),
		)
	}

	return file, file, nil
}

// getLogDir returns the OS-appropriate per-user log directory.
func getLogDir(appName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errorc.With(
			errorspkg.ErrCannotResolveUserHomeDir,
			errorc.Error(keys.Cause, err),
		)
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", appName), nil

	case "windows":
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, appName), nil
		}
		return filepath.Join(home, "AppData", "Roaming", appName), nil

	default:
		if v := os.Getenv("XDG_DATA_HOME"); v != "" && filepath.IsAbs(v) {
			return filepath.Join(v, appName), nil
		}

		return filepath.Join(home, ".local", "share", appName), nil
	}
}
