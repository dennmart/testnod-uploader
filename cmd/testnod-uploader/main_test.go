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
