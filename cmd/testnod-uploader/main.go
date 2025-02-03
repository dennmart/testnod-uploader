package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type multiStringFlag []string

func main() {
	var (
		token          = flag.String("token", "", "TestNod project token")
		validateFile   = flag.Bool("validate", false, "Checks if the file is a valid JUnit XML file, returns without uploading to TestNod")
		ignoreFailures = flag.Bool("ignore-failures", false, "Always return an exit code of 0 even if there are errors")
		uploadURL      = flag.String("upload-url", "", "Specify a custom upload URL to upload the JUnit XML file to TestNod")
		tags           multiStringFlag
	)

	flag.Var(&tags, "tag", "Add a tag to this test run (can be repeated)")

	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("No file specified")

		if *ignoreFailures {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	filePath := args[0]

	if *validateFile {
		fmt.Println("Validating file:", filePath)
		// TODO: Validate file
		os.Exit(0)
	}

	if *token == "" {
		fmt.Println("No token specified")
		if *ignoreFailures {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	testNodUploadURL := "https://testnod.com/upload"
	if *uploadURL != "" {
		testNodUploadURL = *uploadURL
	}

	fmt.Printf("Uploading %s to %s using project token %s...\n", filePath, testNodUploadURL, *token)

	if len(tags) > 0 {
		fmt.Println("Tags:", tags)
	}
	// TODO: Upload file
}

func (m *multiStringFlag) String() string {
	return strings.Join(*m, ", ")
}

func (m *multiStringFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}
