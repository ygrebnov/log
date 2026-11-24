package fs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ygrebnov/errorc"
	sharedfs "github.com/ygrebnov/fs"
	fspath "github.com/ygrebnov/fs/path"
	"github.com/ygrebnov/log/pkg/config"
	errorspkg "github.com/ygrebnov/log/pkg/errors"
	"github.com/ygrebnov/log/pkg/keys"
)

type Closer interface {
	Close() error
}

const (
	parentDir = ".."
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
		return nil, nil, errorc.WithCause(errorspkg.ErrCannotResolveUserHomeDir, err)
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

	// Validate the application directory using a handle opened through its root.
	appDir, err := appRoot.Open(".")
	if err != nil {
		return nil, nil, errorc.With(
			errorspkg.ErrCannotAccessLogFilePath,
			errorc.String(keys.LogDir, logDir),
			errorc.Error(keys.Cause, err),
		)
	}
	defer appDir.Close()

	if errPerm := sharedfs.ValidatePrivateDirectory(appDir); errPerm != nil {
		return nil, nil, errorc.With(
			errorspkg.ErrCannotAccessLogFilePath,
			errorc.String(keys.LogDir, logDir),
			errorc.Error(keys.Cause, logPermissionError(errPerm)),
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
	file, err := sharedfs.OpenRegularAppendWithinRoot(root, name, 0o600)
	if err != nil {
		if errors.Is(err, sharedfs.ErrNotRegularFile) {
			return nil, nil, errorc.With(
				errorspkg.ErrCannotAccessLogFilePath,
				errorc.String(keys.LogFilePath, displayPath),
			)
		}
		return nil, nil, errorc.With(
			errorspkg.ErrCannotAccessLogFilePath,
			errorc.String(keys.LogFilePath, displayPath),
			errorc.Error(keys.Cause, err),
		)
	}

	// The append helper has already checked the opened file type.
	if errPerm := sharedfs.ValidatePrivateFile(file); errPerm != nil {
		_ = file.Close()
		if errors.Is(errPerm, sharedfs.ErrNotRegularFile) {
			return nil, nil, errorc.With(
				errorspkg.ErrCannotAccessLogFilePath,
				errorc.String(keys.LogFilePath, displayPath),
			)
		}

		return nil, nil, errorc.With(
			errorspkg.ErrCannotAccessLogFilePath,
			errorc.String(keys.LogFilePath, displayPath),
			errorc.Error(keys.Cause, logPermissionError(errPerm)),
		)
	}

	return file, file, nil
}

func logPermissionError(err error) error {
	if errors.Is(err, sharedfs.ErrInsecurePermissions) {
		return errorc.WithCause(errorspkg.ErrInsecurePermissions, err)
	}
	return err
}

// getLogDir returns the OS-appropriate per-user log directory.
func getLogDir(appName string) (string, error) {
	dir, err := fspath.DataDir(appName)
	if err != nil {
		switch {
		case errors.Is(err, fspath.ErrInvalidAppName):
			return "", errorspkg.ErrInvalidAppName
		case errors.Is(err, fspath.ErrCannotResolveHome):
			return "", errorc.WithCause(errorspkg.ErrCannotResolveUserHomeDir, err)
		default:
			return "", errorc.WithCause(errorspkg.ErrCannotAccessLogFilePath, err)
		}
	}

	if errValidate := fspath.ValidateWithinHome(dir); errValidate != nil {
		if errors.Is(errValidate, fspath.ErrCannotResolveHome) {
			return "", errorc.WithCause(errorspkg.ErrCannotResolveUserHomeDir, errValidate)
		}

		return "", errorc.With(
			errorspkg.ErrCannotAccessLogFilePath,
			errorc.String(keys.LogDir, dir),
			errorc.Error(keys.Cause, errValidate),
		)
	}

	return dir, nil
}
