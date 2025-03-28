package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"testnod-uploader/internal/testnod"
	"testnod-uploader/internal/upload"
	"testnod-uploader/internal/validation"
)

type uploadTagsFlag []testnod.Tag

func main() {
	var (
		token          = flag.String("token", "", "TestNod project token")
		validateFile   = flag.Bool("validate", false, "Checks if the file is a valid JUnit XML file, returns without uploading to TestNod")
		branch         = flag.String("branch", "", "The branch name used for this test run")
		commitSHA      = flag.String("commit-sha", "", "The commit SHA used for this test run")
		runURL         = flag.String("run-url", "", "The URL to the CI/CD run")
		buildID        = flag.String("build-id", "", "The build identifier for the CI/CD run")
		ignoreFailures = flag.Bool("ignore-failures", false, "Always return an exit code of 0 even if there are errors")
		uploadURL      = flag.String("upload-url", "", "Specify a custom upload URL to upload the JUnit XML file to TestNod")
		tags           uploadTagsFlag
	)

	flag.Var(&tags, "tag", "Add a tag to this test run (can be repeated)")

	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("No file specified")
		exitBasedOnIgnoreFailures(*ignoreFailures)
	}

	filePath := args[0]
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("File not found: %s\n", filePath)
		exitBasedOnIgnoreFailures(*ignoreFailures)
	}

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

	testNodUploadURL := "https://testnod.com/platform/test_runs/upload"
	if *uploadURL != "" {
		testNodUploadURL = *uploadURL
	}

	fmt.Printf("%s is a valid JUnit XML file. Creating test run...\n", filePath)

	uploadRequest := testnod.CreateTestRunRequest{
		Tags: tags,
		TestRun: testnod.TestRun{
			Metadata: testnod.TestRunMetadata{
				Branch:    *branch,
				CommitSHA: *commitSHA,
				RunURL:    *runURL,
				BuildID:   *buildID,
			},
		},
	}

	serverResponse, err := testnod.CreateTestRun(testNodUploadURL, *token, uploadRequest)
	if err != nil {
		fmt.Println("There was an error creating the test run on TestNod.")
		exitBasedOnIgnoreFailures(*ignoreFailures)
	}

	fmt.Println("Created test run, uploading JUnit XML file...")
	err = upload.UploadJUnitXmlFile(filePath, serverResponse.PresignedURL)

	if err != nil {
		fmt.Println("There was an error uploading the file to TestNod. We've been notified and will look into it. Sorry for the inconvenience.")
		exitBasedOnIgnoreFailures(*ignoreFailures)
	}

	fmt.Printf("Test run uploaded successfully! TestNod will now process your test run. You can follow its progress at %s\n", serverResponse.TestRunURL)
	os.Exit(0)
}

func (m *uploadTagsFlag) String() string {
	var values []string
	for _, tag := range *m {
		values = append(values, tag.Value)
	}
	return strings.Join(values, ",")
}

func (m *uploadTagsFlag) Set(value string) error {
	*m = append(*m, testnod.Tag{Value: value})
	return nil
}

func exitBasedOnIgnoreFailures(ignoreFailures bool) {
	if ignoreFailures {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
