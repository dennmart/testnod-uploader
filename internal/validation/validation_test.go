package validation

import (
	"os"
	"strings"
	"testing"
)

func TestValidateJUnitXMLFile(t *testing.T) {
	tests := []struct {
		name     string
		xmlData  string
		wantErr  bool
		errMatch string
	}{
		{
			name: "valid junit xml with testsuite",
			xmlData: `<?xml version="1.0" encoding="UTF-8"?>
<testsuites>
	<testsuite name="test.example" tests="1" failures="0" errors="0" time="0.001">
		<testcase name="test_example" classname="test.example" time="0.001"/>
	</testsuite>
</testsuites>`,
			wantErr: false,
		},
		{
			name: "valid junit xml with single testsuite",
			xmlData: `<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="test.example" tests="1" failures="0" errors="0" time="0.001">
	<testcase name="test_example" classname="test.example" time="0.001"/>
</testsuite>`,
			wantErr: false,
		},
		{
			name: "xml without testsuite element",
			xmlData: `<?xml version="1.0" encoding="UTF-8"?>
<root>
	<testcase name="test_example" classname="test.example" time="0.001"/>
</root>`,
			wantErr:  true,
			errMatch: "doesn't seem to be a valid JUnit XML file",
		},
		{
			name: "empty xml",
			xmlData: `<?xml version="1.0" encoding="UTF-8"?>
<root></root>`,
			wantErr:  true,
			errMatch: "doesn't seem to be a valid JUnit XML file",
		},
		{
			name:     "malformed xml",
			xmlData:  `<?xml version="1.0" encoding="UTF-8"?><testsuite><unclosed>`,
			wantErr:  true,
			errMatch: "error parsing XML",
		},
		{
			name:     "invalid xml characters",
			xmlData:  `<?xml version="1.0" encoding="UTF-8"?><testsuite>` + string(rune(0x00)) + `</testsuite>`,
			wantErr:  true,
			errMatch: "error parsing XML",
		},
		{
			name: "nested testsuite elements",
			xmlData: `<?xml version="1.0" encoding="UTF-8"?>
<testsuites>
	<testsuite name="outer">
		<testsuite name="inner">
			<testcase name="test_example" classname="test.example" time="0.001"/>
		</testsuite>
	</testsuite>
</testsuites>`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile, err := os.CreateTemp("", "junit_test_*.xml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write test data
			if _, err := tmpFile.WriteString(tt.xmlData); err != nil {
				t.Fatalf("Failed to write test data: %v", err)
			}
			tmpFile.Close()

			// Test validation
			err = ValidateJUnitXMLFile(tmpFile.Name())

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateJUnitXMLFile() expected error but got none")
					return
				}
				if tt.errMatch != "" && !strings.Contains(err.Error(), tt.errMatch) {
					t.Errorf("ValidateJUnitXMLFile() error = %v, expected to contain %q", err, tt.errMatch)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateJUnitXMLFile() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateJUnitXMLFileErrors(t *testing.T) {
	t.Run("file not found", func(t *testing.T) {
		err := ValidateJUnitXMLFile("/path/that/does/not/exist.xml")
		if err == nil {
			t.Error("ValidateJUnitXMLFile() expected error for non-existent file")
		}
		if !strings.Contains(err.Error(), "failed to open file") {
			t.Errorf("ValidateJUnitXMLFile() error = %v, expected to contain 'failed to open file'", err)
		}
	})

	t.Run("directory instead of file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "junit_test_dir")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		err = ValidateJUnitXMLFile(tmpDir)
		if err == nil {
			t.Error("ValidateJUnitXMLFile() expected error for directory")
		}
	})

	t.Run("permission denied", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping permission test when running as root")
		}

		tmpFile, err := os.CreateTemp("", "junit_test_*.xml")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		tmpFile.WriteString(`<?xml version="1.0"?><testsuite></testsuite>`)
		tmpFile.Close()

		// Remove read permissions
		if err := os.Chmod(tmpFile.Name(), 0000); err != nil {
			t.Fatalf("Failed to change file permissions: %v", err)
		}
		defer os.Chmod(tmpFile.Name(), 0644) // Restore permissions for cleanup

		err = ValidateJUnitXMLFile(tmpFile.Name())
		if err == nil {
			t.Error("ValidateJUnitXMLFile() expected error for permission denied")
		}
	})
}

func TestValidateJUnitXMLFileWithRealExamples(t *testing.T) {
	examples := []struct {
		name    string
		content string
		valid   bool
	}{
		{
			name: "gradle test output",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="com.example.MyTest" tests="3" skipped="0" failures="1" errors="0" timestamp="2023-01-01T12:00:00" hostname="localhost" time="0.123">
  <properties/>
  <testcase name="testSuccess" classname="com.example.MyTest" time="0.001"/>
  <testcase name="testFailure" classname="com.example.MyTest" time="0.002">
    <failure message="Expected true but was false" type="java.lang.AssertionError">
      java.lang.AssertionError: Expected true but was false
      at com.example.MyTest.testFailure(MyTest.java:15)
    </failure>
  </testcase>
  <testcase name="testSkipped" classname="com.example.MyTest" time="0.000">
    <skipped/>
  </testcase>
  <system-out><![CDATA[]]></system-out>
  <system-err><![CDATA[]]></system-err>
</testsuite>`,
			valid: true,
		},
		{
			name: "maven surefire output",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<testsuites>
  <testsuite name="com.example.FirstTest" time="0.050" tests="2" errors="0" skipped="0" failures="0">
    <testcase name="test1" classname="com.example.FirstTest" time="0.025"/>
    <testcase name="test2" classname="com.example.FirstTest" time="0.025"/>
  </testsuite>
  <testsuite name="com.example.SecondTest" time="0.030" tests="1" errors="0" skipped="0" failures="0">
    <testcase name="test3" classname="com.example.SecondTest" time="0.030"/>
  </testsuite>
</testsuites>`,
			valid: true,
		},
		{
			name: "pytest junit output",
			content: `<?xml version="1.0" encoding="utf-8"?>
<testsuites>
  <testsuite name="pytest" errors="0" failures="0" skipped="0" tests="1" time="0.001" timestamp="2023-01-01T12:00:00.000000" hostname="localhost">
    <testcase classname="test_example" name="test_function" time="0.001"/>
  </testsuite>
</testsuites>`,
			valid: true,
		},
	}

	for _, example := range examples {
		t.Run(example.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "junit_real_*.xml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(example.content); err != nil {
				t.Fatalf("Failed to write test data: %v", err)
			}
			tmpFile.Close()

			err = ValidateJUnitXMLFile(tmpFile.Name())
			if example.valid && err != nil {
				t.Errorf("ValidateJUnitXMLFile() unexpected error for valid example: %v", err)
			} else if !example.valid && err == nil {
				t.Errorf("ValidateJUnitXMLFile() expected error for invalid example")
			}
		})
	}
}
