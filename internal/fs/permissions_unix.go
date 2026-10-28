//go:build !windows

package fs

import (
	"os"

	"github.com/ygrebnov/errorc"
	"github.com/ygrebnov/log/pkg/errors"
)

func validatePrivatePermissions(file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}

	if info.Mode().Perm()&0o077 != 0 {
		return errorc.With(
			errors.ErrInsecurePermissions,
			errorc.String("validation", "insecure file permissions"),
		)
	}

	return nil
}

func validatePrivateDirectoryPermissions(root *os.Root) error {
	info, err := root.Stat(".")
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return errorc.With(
			errors.ErrInsecurePermissions,
			errorc.String("validation", "application root is not a directory"),
		)
	}

	if info.Mode().Perm()&0o077 != 0 {
		return errorc.With(
			errors.ErrInsecurePermissions,
			errorc.String("validation", "application directory has insecure permissions"),
		)
	}

	return nil
}
