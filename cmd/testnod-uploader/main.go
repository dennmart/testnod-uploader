package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"testnod-uploader/internal/upload"
	"testnod-uploader/internal/validation"
)

type multiStringFlag []string

func main() {
	var (
		token          = flag.String("token", "", "TestNod project token")
		validateFile   = flag.Bool("validate", false, "Checks if the file is a valid JUnit XML file, returns without uploading to TestNod")
		branch         = flag.String("branch", "", "The branch name used for this test run")
		commitSHA      = flag.String("commit-sha", "", "The commit SHA used for this test run")
		runURL         = flag.String("run-url", "", "The URL to the CI/CD run")
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

		err := validation.ValidateJUnitXMLFile(filePath)
		if err != nil {
			fmt.Println(err)
			exitBasedOnIgnoreFailures(*ignoreFailures)
		}

		fmt.Printf("%s is a valid JUnit XML file!\n", filePath)
		os.Exit(0)
	}

	if *token == "" {
		fmt.Println("No token specified")
		exitBasedOnIgnoreFailures(*ignoreFailures)
	}

	testNodUploadURL := "https://testnod.com/test_runs/upload"
	if *uploadURL != "" {
		testNodUploadURL = *uploadURL
	}

	fmt.Printf("%s is a valid JUnit XML file. Uploading file for processing on TestNod...\n", filePath)

	uploadRequest := upload.UploadRequest{
		TestRun: upload.TestRun{
			Metadata: upload.TestRunMetadata{
				Branch:    *branch,
				CommitSHA: *commitSHA,
				RunURL:    *runURL,
				Tags:      tags,
			},
		},
	}

	statusCode, testRunUrl, err := upload.UploadJUnitXmlFile(filePath, testNodUploadURL, *token, uploadRequest)

	if err != nil {
		if statusCode == 0 {
			fmt.Println("There was an error uploading the file to TestNod. We've been notified and will look into it. Sorry for the inconvenience.")
			exitBasedOnIgnoreFailures(*ignoreFailures)
		} else {
			fmt.Printf("TestNod returned an error: %s\n", err)
			exitBasedOnIgnoreFailures(*ignoreFailures)
		}
	}

	fmt.Printf("Test run uploaded successfully! TestNod will now process your test run. You can follow its progress at %s\n", testRunUrl)
	os.Exit(0)
}

func (m *multiStringFlag) String() string {
	return strings.Join(*m, ", ")
}

func (m *multiStringFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func exitBasedOnIgnoreFailures(ignoreFailures bool) {
	if ignoreFailures {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
