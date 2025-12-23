package validation

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
)

func ValidateJUnitXMLFile(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	decoder := xml.NewDecoder(f)

	var hasTestSuite bool

	for {
		t, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("error parsing XML: %w", err)
		}

		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "testsuite" {
				hasTestSuite = true
			}
		}
	}

	if !hasTestSuite {
		return fmt.Errorf("doesn't seem to be a valid JUnit XML file")
	}

	return nil
}
