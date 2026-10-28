package fs

import (
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

const (
	parentDir = ".."
	osWindows = "windows"
)

// OpenWriter opens a writer for the configured log destination.
// On success, the caller owns the returned Closer and must close it.
func OpenWriter(cfg *config.SinkConfig, appName string) (io.Writer, Closer, error) {
	path := strings.TrimSpace(cfg.Path)

	if path != "" {
		// An explicit path is authorized by the application.
		// Its parent directory is the confinement boundary.
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, nil, errorc.With(
				errorspkg.ErrCannotAccessLogFilePath,
				errorc.String(keys.LogFilePath, path),
				errorc.Error(keys.Cause, err),
			)
		}

		root, err := os.OpenRoot(filepath.Dir(path))
		if err != nil {
			return nil, nil, errorc.With(
				errorspkg.ErrCannotAccessLogFilePath,
				errorc.String(keys.LogFilePath, path),
				errorc.Error(keys.Cause, err),
			)
		}
		defer root.Close()

		return openFile(root, filepath.Base(path), path)
	}

	logDir, err := getLogDir(appName)
	if err != nil {
		return nil, nil, err
	}

	// The OS-provided home directory is the trust boundary for
	// the default destination. Subsequent traversal stays beneath it.
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, errorc.With(
			errorspkg.ErrCannotResolveUserHomeDir,
			errorc.Error(keys.Cause, err),
		)
	}

	homeRoot, err := os.OpenRoot(home)
	if err != nil {
		return nil, nil, errorc.With(
			errorspkg.ErrCannotAccessLogFilePath,
			errorc.String(keys.LogDir, logDir),
			errorc.Error(keys.Cause, err),
		)
	}
	defer homeRoot.Close()

	relDir, err := filepath.Rel(home, logDir)
	if err != nil ||
		relDir == parentDir ||
		filepath.IsAbs(relDir) ||
		strings.HasPrefix(relDir, parentDir+string(filepath.Separator)) {
		return nil, nil, errorc.With(
			errorspkg.ErrCannotAccessLogFilePath,
			errorc.String(keys.LogDir, logDir),
		)
	}

	if errMkdir := homeRoot.MkdirAll(relDir, 0o700); errMkdir != nil {
		return nil, nil, errorc.With(
			errorspkg.ErrCannotAccessLogFilePath,
			errorc.String(keys.LogDir, logDir),
			errorc.Error(keys.Cause, errMkdir),
		)
	}

	appRoot, err := homeRoot.OpenRoot(relDir)
	if err != nil {
		return nil, nil, errorc.With(
			errorspkg.ErrCannotAccessLogFilePath,
			errorc.String(keys.LogDir, logDir),
			errorc.Error(keys.Cause, err),
		)
	}
	defer appRoot.Close()

	// Validate the application directory using its opened root handle.
	// The platform-specific helper checks Unix mode bits or Windows ACLs.
	if errPerm := validatePrivateDirectoryPermissions(appRoot); errPerm != nil {
		return nil, nil, errorc.With(
			errorspkg.ErrCannotAccessLogFilePath,
			errorc.String(keys.LogDir, logDir),
			errorc.Error(keys.Cause, errPerm),
		)
	}

	return openFile(
		appRoot,
		appName+".log",
		filepath.Join(logDir, appName+".log"),
	)
}

// openFile opens a log file within an established filesystem root.
// On success, the caller owns the returned Closer and must close it.
func openFile(root *os.Root, name, displayPath string) (io.Writer, Closer, error) {
	// Reject existing non-regular files before opening.
	// This check is not atomic with the subsequent OpenFile call.
	if info, err := root.Stat(name); err == nil {
		if !info.Mode().IsRegular() {
			return nil, nil, errorc.With(
				errorspkg.ErrCannotAccessLogFilePath,
				errorc.String(keys.LogFilePath, displayPath),
			)
		}
	} else if !os.IsNotExist(err) {
		return nil, nil, errorc.With(
			errorspkg.ErrCannotAccessLogFilePath,
			errorc.String(keys.LogFilePath, displayPath),
			errorc.Error(keys.Cause, err),
		)
	}

	file, err := root.OpenFile(
		name,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0o600,
	)
	if err != nil {
		return nil, nil, errorc.With(
			errorspkg.ErrCannotAccessLogFilePath,
			errorc.String(keys.LogFilePath, displayPath),
			errorc.Error(keys.Cause, err),
		)
	}

	// Verify the opened file rather than relying solely on
	// the preliminary path-based check.
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()

		return nil, nil, errorc.With(
			errorspkg.ErrCannotAccessLogFilePath,
			errorc.String(keys.LogFilePath, displayPath),
			errorc.Error(keys.Cause, err),
		)
	}

	if !info.Mode().IsRegular() {
		_ = file.Close()

		return nil, nil, errorc.With(
			errorspkg.ErrCannotAccessLogFilePath,
			errorc.String(keys.LogFilePath, displayPath),
		)
	}

	if errPerm := validatePrivatePermissions(file); errPerm != nil {
		_ = file.Close()

		return nil, nil, errorc.With(
			errorspkg.ErrCannotAccessLogFilePath,
			errorc.String(keys.LogFilePath, displayPath),
			errorc.Error(keys.Cause, errPerm),
		)
	}

	return file, file, nil
}

// getLogDir returns the OS-appropriate per-user log directory.
func getLogDir(appName string) (string, error) {
	if err := validateAppName(appName); err != nil {
		return "", err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", errorc.With(
			errorspkg.ErrCannotResolveUserHomeDir,
			errorc.Error(keys.Cause, err),
		)
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(
			home,
			"Library",
			"Application Support",
			appName,
		), nil

	case osWindows:
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			if err := validateBaseWithinHome(home, appdata); err != nil {
				return "", err
			}

			return filepath.Join(appdata, appName), nil
		}

		return filepath.Join(home, "AppData", "Roaming", appName), nil

	default:
		if v := os.Getenv("XDG_DATA_HOME"); v != "" && filepath.IsAbs(v) {
			if err := validateBaseWithinHome(home, v); err != nil {
				return "", err
			}

			return filepath.Join(v, appName), nil
		}

		return filepath.Join(home, ".local", "share", appName), nil
	}
}

func validateAppName(appName string) error {
	if appName == "" ||
		appName == "." ||
		appName == ".." ||
		strings.TrimSpace(appName) != appName ||
		strings.ContainsAny(appName, `/\`) ||
		filepath.IsAbs(appName) ||
		strings.ContainsAny(appName, `<>:"|?*`) ||
		strings.HasSuffix(appName, ".") {
		return errorspkg.ErrInvalidAppName
	}

	for _, r := range appName {
		if r < 0x20 || r == 0x7f {
			return errorspkg.ErrInvalidAppName
		}
	}

	// Windows reserves certain device names, even when followed
	// by an extension. The comparison is case-insensitive.
	base := strings.ToUpper(strings.SplitN(appName, ".", 2)[0])

	switch base {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
		return errorspkg.ErrInvalidAppName
	}

	if len(base) == 4 &&
		(strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) {
		if base[3] >= '1' && base[3] <= '9' {
			return errorspkg.ErrInvalidAppName
		}
	}

	return nil
}

// validateBaseWithinHome requires an environment-provided data directory
// to be lexically inside the user's home directory. This is a path check,
// not a symlink or directory-identity check.
func validateBaseWithinHome(home, base string) error {
	if !filepath.IsAbs(base) {
		return errorc.With(
			errorspkg.ErrCannotAccessLogFilePath,
			errorc.String(keys.LogDir, base),
		)
	}

	rel, err := filepath.Rel(home, base)
	if err != nil ||
		rel == parentDir ||
		filepath.IsAbs(rel) ||
		strings.HasPrefix(rel, parentDir+string(filepath.Separator)) {
		return errorc.With(
			errorspkg.ErrCannotAccessLogFilePath,
			errorc.String(keys.LogDir, base),
			errorc.Error(keys.Cause, err),
		)
	}

	return nil
}
