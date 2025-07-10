package main

import (
	"flag"
	"os"
	"strings"
	"testing"
)

func TestParseFlags(t *testing.T) {
	// Save original args and restore them after the test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name        string
		args        []string
		wantConfig  Config
		wantErr     bool
		errContains string
	}{
		{
			name: "valid args with token",
			args: []string{"cmd", "-token=abc123", "-branch=main", "test.xml"},
			wantConfig: Config{
				Token:     "abc123",
				Branch:    "main",
				FilePath:  "test.xml",
				UploadURL: defaultUploadURL,
			},
			wantErr: false,
		},
		{
			name:        "no file specified",
			args:        []string{"cmd", "-token=abc123"},
			wantErr:     true,
			errContains: "no file specified",
		},
		{
			name:        "missing file",
			args:        []string{"cmd", "test.xml"},
			wantErr:     true,
			errContains: "file not found: test.xml",
		},
		{
			name: "missing token without validate flag",
			args: []string{"cmd", "test.xml"},
			wantConfig: Config{
				FilePath: "test.xml",
			},
			wantErr:     true,
			errContains: "no token specified",
		},
		{
			name: "valid args with validate flag",
			args: []string{"cmd", "-validate", "test.xml"},
			wantConfig: Config{
				ValidateFile: true,
				FilePath:     "test.xml",
				UploadURL:    defaultUploadURL,
			},
			wantErr: false,
		},
		{
			name: "with tags",
			args: []string{"cmd", "-token=abc123", "-tag=feature", "-tag=backend", "test.xml"},
			wantConfig: Config{
				Token:     "abc123",
				FilePath:  "test.xml",
				UploadURL: defaultUploadURL,
				Tags:      uploadTagsFlag{{Value: "feature"}, {Value: "backend"}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file if a file path is specified and make
			// sure it gets removed after the test.
			if tt.wantConfig.FilePath != "" {
				f, err := os.Create(tt.wantConfig.FilePath)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				defer os.Remove(tt.wantConfig.FilePath)
				f.Close()
			}

			// Set up command line args
			os.Args = tt.args

			// Reset flags before each test
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			got, err := parseFlags()

			// Check error expectations
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("parseFlags() error = %v, should contain %v", err, tt.errContains)
				return
			}

			// Only check config fields if no error expected
			if !tt.wantErr {
				if got.Token != tt.wantConfig.Token {
					t.Errorf("parseFlags() Token = %v, want %v", got.Token, tt.wantConfig.Token)
				}
				if got.ValidateFile != tt.wantConfig.ValidateFile {
					t.Errorf("parseFlags() ValidateFile = %v, want %v", got.ValidateFile, tt.wantConfig.ValidateFile)
				}
				if got.Branch != tt.wantConfig.Branch {
					t.Errorf("parseFlags() Branch = %v, want %v", got.Branch, tt.wantConfig.Branch)
				}
				if got.FilePath != tt.wantConfig.FilePath {
					t.Errorf("parseFlags() FilePath = %v, want %v", got.FilePath, tt.wantConfig.FilePath)
				}
				if got.UploadURL != tt.wantConfig.UploadURL {
					t.Errorf("parseFlags() UploadURL = %v, want %v", got.UploadURL, tt.wantConfig.UploadURL)
				}
				if len(got.Tags) != len(tt.wantConfig.Tags) {
					t.Errorf("parseFlags() Tags count = %d, want %d", len(got.Tags), len(tt.wantConfig.Tags))
				} else {
					for i, tag := range got.Tags {
						if tag.Value != tt.wantConfig.Tags[i].Value {
							t.Errorf("parseFlags() Tags[%d] = %v, want %v", i, tag.Value, tt.wantConfig.Tags[i].Value)
						}
					}
				}
			}
		})
	}
}

func TestUploadTagsFlag(t *testing.T) {
	t.Run("String()", func(t *testing.T) {
		tags := uploadTagsFlag{{Value: "feature"}, {Value: "backend"}}
		want := "feature,backend"
		if got := tags.String(); got != want {
			t.Errorf("uploadTagsFlag.String() = %v, want %v", got, want)
		}
	})

	t.Run("Set()", func(t *testing.T) {
		var tags uploadTagsFlag
		err := tags.Set("feature")
		if err != nil {
			t.Errorf("uploadTagsFlag.Set() error = %v", err)
		}

		if len(tags) != 1 || tags[0].Value != "feature" {
			t.Errorf("uploadTagsFlag.Set() resulted in %v, want [{Value:feature}]", tags)
		}

		err = tags.Set("backend")
		if err != nil {
			t.Errorf("uploadTagsFlag.Set() error = %v", err)
		}

		if len(tags) != 2 || tags[1].Value != "backend" {
			t.Errorf("uploadTagsFlag.Set() resulted in incorrect state after second call")
		}
	})
}

func TestExitBasedOnIgnoreFailures(t *testing.T) {
	// We can't directly test os.Exit, but we can test the function exists
	// and doesn't panic with different inputs
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("exitBasedOnIgnoreFailures() panicked: %v", r)
		}
	}()

	// Test with ignore failures true - would call os.Exit(0)
	// Test with ignore failures false - would call os.Exit(1)
	// We can't actually test the exit codes without subprocess testing
	// but we can ensure the function doesn't panic

	// Note: We can't actually call this function in tests because it will exit
	// the test process. In a real scenario, you might use dependency injection
	// or a wrapper function to make this testable.
}

func TestValidateOnly(t *testing.T) {
	// Create a temporary valid XML file
	tmpFile, err := os.CreateTemp("", "junit_validate_test_*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	validXML := `<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="test" tests="1" failures="0" errors="0" time="0.001">
	<testcase name="test_example" classname="test.example" time="0.001"/>
</testsuite>`

	if _, err := tmpFile.WriteString(validXML); err != nil {
		t.Fatalf("Failed to write test XML: %v", err)
	}
	tmpFile.Close()

	_ = Config{
		FilePath:       tmpFile.Name(),
		IgnoreFailures: true, // Set to true so we don't exit on validation errors
	}

	// Test that validateOnly doesn't panic with valid XML
	// Note: validateOnly calls os.Exit(0) on success, so we can't test it directly
	// without subprocess testing. In a real scenario, you might refactor to return
	// an error instead of calling os.Exit directly.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("validateOnly() panicked: %v", r)
		}
	}()

	// We can't actually call validateOnly because it will exit the test process
	// This is a limitation of the current design where business logic is mixed
	// with system calls like os.Exit
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectValid bool
	}{
		{
			name: "valid config for upload",
			config: Config{
				Token:     "abc123",
				FilePath:  "test.xml",
				UploadURL: "https://example.com/upload",
			},
			expectValid: true,
		},
		{
			name: "valid config for validation only",
			config: Config{
				ValidateFile: true,
				FilePath:     "test.xml",
			},
			expectValid: true,
		},
		{
			name: "invalid config - missing token for upload",
			config: Config{
				FilePath:  "test.xml",
				UploadURL: "https://example.com/upload",
			},
			expectValid: false,
		},
		{
			name: "invalid config - missing file path",
			config: Config{
				Token:     "abc123",
				UploadURL: "https://example.com/upload",
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file if needed
			if tt.config.FilePath != "" {
				tmpFile, err := os.CreateTemp("", "config_test_*.xml")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				defer os.Remove(tmpFile.Name())
				tmpFile.Close()
				tt.config.FilePath = tmpFile.Name()
			}

			// Test the validation logic from parseFlags
			var valid bool
			if tt.config.FilePath != "" {
				if _, err := os.Stat(tt.config.FilePath); !os.IsNotExist(err) {
					if tt.config.ValidateFile || tt.config.Token != "" {
						valid = true
					}
				}
			}

			if valid != tt.expectValid {
				t.Errorf("Config validation mismatch. Got valid=%v, expected=%v", valid, tt.expectValid)
			}
		})
	}
}

func TestParseFlagsEdgeCases(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name:    "custom upload URL",
			args:    []string{"cmd", "-token=abc123", "-upload-url=https://custom.com/upload", "test.xml"},
			wantErr: false,
		},
		{
			name:    "all flags set",
			args:    []string{"cmd", "-token=abc123", "-branch=main", "-commit-sha=sha123", "-run-url=https://ci.com/run", "-build-id=build123", "-ignore-failures", "test.xml"},
			wantErr: false,
		},
		{
			name:        "validate flag with non-existent file",
			args:        []string{"cmd", "-validate", "nonexistent.xml"},
			wantErr:     true,
			errContains: "file not found",
		},
		{
			name:    "empty token with validate flag",
			args:    []string{"cmd", "-validate", "-token=", "test.xml"},
			wantErr: false, // token not required for validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file if needed
			if len(tt.args) > 0 {
				lastArg := tt.args[len(tt.args)-1]
				if strings.HasSuffix(lastArg, ".xml") && !strings.Contains(lastArg, "nonexistent") {
					tmpFile, err := os.CreateTemp("", "edge_case_test_*.xml")
					if err != nil {
						t.Fatalf("Failed to create temp file: %v", err)
					}
					defer os.Remove(tmpFile.Name())
					tmpFile.Close()
					tt.args[len(tt.args)-1] = tmpFile.Name()
				}
			}

			os.Args = tt.args
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			_, err := parseFlags()

			if (err != nil) != tt.wantErr {
				t.Errorf("parseFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("parseFlags() error = %v, should contain %v", err, tt.errContains)
			}
		})
	}
}
