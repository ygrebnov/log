package fs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/ygrebnov/log/pkg/config"
	errorspkg "github.com/ygrebnov/log/pkg/errors"
)

// TestWindowsPrivateFileACL verifies that a file with a private DACL
// is accepted by the ACL validator.
func TestWindowsPrivateFileACL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private.log")

	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_RDWR,
		0o600,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	setTestPathACL(t, path, privateACL(t))

	if err := validatePrivatePermissions(file); err != nil {
		t.Fatalf("private ACL rejected: %v", err)
	}
}

// TestWindowsPrivateDirectoryACL verifies that a directory with a
// private DACL is accepted.
func TestWindowsPrivateDirectoryACL(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "private")

	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	setTestPathACL(t, dir, privateACL(t))

	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	if err := validatePrivateDirectoryPermissions(root); err != nil {
		t.Fatalf("private directory ACL rejected: %v", err)
	}
}

// TestWindowsEveryoneFileACL verifies that access granted to
// Everyone causes validation to fail.
func TestWindowsEveryoneFileACL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "public.log")

	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_RDWR,
		0o600,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	sddl := privateACL(t) + "(A;;FR;;;WD)"
	setTestPathACL(t, path, sddl)

	err = validatePrivatePermissions(file)
	if !errors.Is(err, errorspkg.ErrInsecurePermissions) {
		t.Fatalf(
			"expected ErrInsecurePermissions, got: %v",
			err,
		)
	}
}

// TestWindowsUsersDirectoryACL verifies that a directory granting
// access to BUILTIN\Users is rejected.
func TestWindowsUsersDirectoryACL(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "public")

	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	sddl := privateACL(t) + "(A;;FR;;;BU)"
	setTestPathACL(t, dir, sddl)

	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	err = validatePrivateDirectoryPermissions(root)
	if !errors.Is(err, errorspkg.ErrInsecurePermissions) {
		t.Fatalf(
			"expected ErrInsecurePermissions, got: %v",
			err,
		)
	}
}

// TestWindowsNullDACL verifies that the validator rejects a NULL DACL.
//
// A NULL DACL permits unrestricted access and must not be confused
// with an empty DACL, which denies access.
func TestWindowsNullDACL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "null-dacl.log")

	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_RDWR,
		0o600,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	if err := windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION,
		nil,
		nil,
		nil,
		nil,
	); err != nil {
		t.Fatalf("SetNamedSecurityInfo: %v", err)
	}

	// Verify that Windows actually applied a NULL DACL.
	sd, err := windows.GetSecurityInfo(
		windows.Handle(file.Fd()),
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		t.Fatalf("GetSecurityInfo: %v", err)
	}

	dacl, _, err := sd.DACL()
	if err != nil {
		t.Fatalf("DACL: %v", err)
	}

	if dacl != nil {
		t.Fatal("expected NULL DACL")
	}

	// Verify that the validator rejects it.
	err = validatePrivatePermissions(file)
	if !errors.Is(err, errorspkg.ErrInsecurePermissions) {
		t.Fatalf(
			"expected ErrInsecurePermissions, got: %v",
			err,
		)
	}
}

// TestWindowsInheritedACL verifies that a file with its default
// inherited ACL is accepted.
//
// This expectation was verified on our Windows test machine.
// Other Windows environments may have different inherited ACLs.
func TestWindowsInheritedACL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "inherited.log")

	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_RDWR,
		0o600,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	if err := validatePrivatePermissions(file); err != nil {
		t.Fatalf("inherited ACL rejected: %v", err)
	}
}

// TestWindowsDefaultApplicationDirectory exercises the complete
// default-destination initialization path using an isolated APPDATA directory.
func TestWindowsDefaultApplicationDirectory(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	base, err := os.MkdirTemp(home, "log-appdata-test-")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(base); err != nil {
			t.Errorf("cleanup failed: %v", err)
		}
	})

	t.Setenv("APPDATA", base)

	appName := "logger-acl-test"

	writer, closer, err := OpenWriter(&config.SinkConfig{}, appName)
	if err != nil {
		t.Fatalf("OpenWriter() failed: %v", err)
	}

	if _, err := writer.Write([]byte("ACL integration test\n")); err != nil {
		_ = closer.Close()
		t.Fatalf("Write() failed: %v", err)
	}

	if err := closer.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	path := filepath.Join(base, appName, appName+".log")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	if string(content) != "ACL integration test\n" {
		t.Fatalf("unexpected log content: %q", content)
	}
}

// privateACL returns an SDDL string granting full access only to
// the current user, SYSTEM, and Administrators.
func privateACL(t *testing.T) string {
	t.Helper()

	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}

	return "D:P" +
		"(A;;FA;;;SY)" +
		"(A;;FA;;;BA)" +
		"(A;;FA;;;" + user.User.Sid.String() + ")"
}

// setTestPathACL applies an SDDL DACL to a file or directory.
//
// We use SetNamedSecurityInfo rather than SetSecurityInfo because
// modifying the ACL through an os.File handle produced access-denied
// errors on Windows.
func setTestPathACL(t *testing.T, path, sddl string) {
	t.Helper()

	sd, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		t.Fatalf("SecurityDescriptorFromString: %v", err)
	}

	dacl, _, err := sd.DACL()
	if err != nil {
		t.Fatalf("DACL: %v", err)
	}

	if err := windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|
			windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		dacl,
		nil,
	); err != nil {
		t.Fatalf("SetNamedSecurityInfo: %v", err)
	}
}
