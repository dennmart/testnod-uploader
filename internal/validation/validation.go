package validation

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"

	"testnod-uploader/internal/debug"
)

func ValidateJUnitXMLFile(filePath string) error {
	debug.Log("validating file: %s", filePath)
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	decoder := xml.NewDecoder(f)

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
			if se.Name.Local == "testsuite" || se.Name.Local == "testsuites" {
				debug.Log("found valid root element: <%s>", se.Name.Local)
				return nil
			}
		}
	}

	return fmt.Errorf("file does not contain a <testsuite> or <testsuites> element")
}
