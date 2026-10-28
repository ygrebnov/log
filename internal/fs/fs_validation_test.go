package fs

import (
	"errors"
	"testing"

	errorspkg "github.com/ygrebnov/log/pkg/errors"
)

func TestValidateAppName(t *testing.T) {
	tests := []struct {
		name    string
		appName string
		wantErr bool
	}{
		// Valid application names.
		{
			name:    "simple",
			appName: "e2engine",
		},
		{
			name:    "hyphen",
			appName: "e2engine-cloud",
		},
		{
			name:    "underscore",
			appName: "my_application",
		},
		{
			name:    "dots",
			appName: "com.example.app",
		},
		{
			name:    "spaces",
			appName: "My Application",
		},
		{
			name:    "unicode",
			appName: "Моё приложение",
		},
		{
			name:    "windows reserved prefix allowed",
			appName: "console",
		},
		{
			name:    "windows device prefix allowed",
			appName: "COM10",
		},
		{
			name:    "windows device zero allowed",
			appName: "COM0",
		},
		{
			name:    "windows printer zero allowed",
			appName: "LPT0",
		},

		// Invalid path components.
		{
			name:    "empty",
			appName: "",
			wantErr: true,
		},
		{
			name:    "current directory",
			appName: ".",
			wantErr: true,
		},
		{
			name:    "parent directory",
			appName: "..",
			wantErr: true,
		},
		{
			name:    "unix traversal",
			appName: "../other",
			wantErr: true,
		},
		{
			name:    "windows traversal",
			appName: `..\other`,
			wantErr: true,
		},
		{
			name:    "unix absolute path",
			appName: "/tmp/logs",
			wantErr: true,
		},
		{
			name:    "windows absolute path",
			appName: `C:\logs`,
			wantErr: true,
		},
		{
			name:    "forward slash",
			appName: "my/app",
			wantErr: true,
		},
		{
			name:    "backslash",
			appName: `my\app`,
			wantErr: true,
		},
		{
			name:    "leading whitespace",
			appName: " app",
			wantErr: true,
		},
		{
			name:    "trailing whitespace",
			appName: "app ",
			wantErr: true,
		},
		{
			name:    "trailing period",
			appName: "app.",
			wantErr: true,
		},

		// Windows invalid characters.
		{
			name:    "colon",
			appName: "app:name",
			wantErr: true,
		},
		{
			name:    "asterisk",
			appName: "app*name",
			wantErr: true,
		},
		{
			name:    "question mark",
			appName: "app?name",
			wantErr: true,
		},
		{
			name:    "double quote",
			appName: `app"name`,
			wantErr: true,
		},
		{
			name:    "less than",
			appName: "app<name",
			wantErr: true,
		},
		{
			name:    "greater than",
			appName: "app>name",
			wantErr: true,
		},
		{
			name:    "pipe",
			appName: "app|name",
			wantErr: true,
		},

		// Control characters.
		{
			name:    "newline",
			appName: "app\nname",
			wantErr: true,
		},
		{
			name:    "tab",
			appName: "app\tname",
			wantErr: true,
		},
		{
			name:    "null byte",
			appName: "app\x00name",
			wantErr: true,
		},
		{
			name:    "delete character",
			appName: "app\x7fname",
			wantErr: true,
		},

		// Windows reserved device names.
		{
			name:    "CON",
			appName: "CON",
			wantErr: true,
		},
		{
			name:    "PRN",
			appName: "PRN",
			wantErr: true,
		},
		{
			name:    "AUX",
			appName: "AUX",
			wantErr: true,
		},
		{
			name:    "NUL",
			appName: "NUL",
			wantErr: true,
		},
		{
			name:    "COM1",
			appName: "COM1",
			wantErr: true,
		},
		{
			name:    "COM9",
			appName: "COM9",
			wantErr: true,
		},
		{
			name:    "LPT1",
			appName: "LPT1",
			wantErr: true,
		},
		{
			name:    "LPT9",
			appName: "LPT9",
			wantErr: true,
		},
		{
			name:    "CONIN$",
			appName: "CONIN$",
			wantErr: true,
		},
		{
			name:    "CONOUT$",
			appName: "CONOUT$",
			wantErr: true,
		},

		// Reserved names with extensions.
		{
			name:    "CON with extension",
			appName: "CON.log",
			wantErr: true,
		},
		{
			name:    "NUL with extension",
			appName: "NUL.txt",
			wantErr: true,
		},
		{
			name:    "COM1 with extension",
			appName: "COM1.log",
			wantErr: true,
		},
		{
			name:    "LPT9 with extension",
			appName: "LPT9.data",
			wantErr: true,
		},

		// Case-insensitive Windows reserved names.
		{
			name:    "lowercase con",
			appName: "con",
			wantErr: true,
		},
		{
			name:    "mixed case com1",
			appName: "CoM1",
			wantErr: true,
		},
		{
			name:    "lowercase nul with extension",
			appName: "nul.log",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAppName(tt.appName)

			if tt.wantErr {
				if !errors.Is(err, errorspkg.ErrInvalidAppName) {
					t.Fatalf(
						"validateAppName(%q) error = %v, want ErrInvalidAppName",
						tt.appName,
						err,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf(
					"validateAppName(%q) unexpected error: %v",
					tt.appName,
					err,
				)
			}
		})
	}
}
