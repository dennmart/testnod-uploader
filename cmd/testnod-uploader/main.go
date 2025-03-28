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

const (
	defaultUploadURL = "https://testnod.com/integrations/test_runs/upload"
)

type Config struct {
	Token          string
	ValidateFile   bool
	Branch         string
	CommitSHA      string
	RunURL         string
	BuildID        string
	IgnoreFailures bool
	UploadURL      string
	Tags           uploadTagsFlag
	FilePath       string
}

func main() {
	config, err := parseFlags()
	if err != nil {
		fmt.Println(err)
		exitBasedOnIgnoreFailures(config.IgnoreFailures)
	}

	if config.ValidateFile {
		validateOnly(config)
		return
	}

	uploadToTestNod(config)
}

func parseFlags() (Config, error) {
	var config Config
	var tags uploadTagsFlag

	flag.StringVar(&config.Token, "token", "", "TestNod project token")
	flag.BoolVar(&config.ValidateFile, "validate", false, "Checks if the file is a valid JUnit XML file, returns without uploading to TestNod")
	flag.StringVar(&config.Branch, "branch", "", "The branch name used for this test run")
	flag.StringVar(&config.CommitSHA, "commit-sha", "", "The commit SHA used for this test run")
	flag.StringVar(&config.RunURL, "run-url", "", "The URL to the CI/CD run")
	flag.StringVar(&config.BuildID, "build-id", "", "The build identifier for the CI/CD run")
	flag.BoolVar(&config.IgnoreFailures, "ignore-failures", false, "Always return an exit code of 0 even if there are errors")
	flag.StringVar(&config.UploadURL, "upload-url", "", "Specify a custom upload URL to upload the JUnit XML file to TestNod")

	flag.Var(&tags, "tag", "Add a tag to this test run (can be repeated)")

	flag.Parse()
	config.Tags = tags

	args := flag.Args()
	if len(args) == 0 {
		return config, fmt.Errorf("no file specified")
	}

	config.FilePath = args[0]
	if _, err := os.Stat(config.FilePath); os.IsNotExist(err) {
		return config, fmt.Errorf("file not found: %s", config.FilePath)
	}

	if config.UploadURL == "" {
		config.UploadURL = defaultUploadURL
	}

	if !config.ValidateFile && config.Token == "" {
		return config, fmt.Errorf("no token specified")
	}

	return config, nil
}

func validateOnly(config Config) {
	fmt.Println("Validating file:", config.FilePath)

	err := validation.ValidateJUnitXMLFile(config.FilePath)
	if err != nil {
		fmt.Println(err)
		exitBasedOnIgnoreFailures(config.IgnoreFailures)
	}

	fmt.Printf("%s is a valid JUnit XML file!\n", config.FilePath)
	os.Exit(0)
}

func uploadToTestNod(config Config) {
	fmt.Printf("%s is a valid JUnit XML file. Creating test run...\n", config.FilePath)

	uploadRequest := testnod.CreateTestRunRequest{
		Tags: config.Tags,
		TestRun: testnod.TestRun{
			Metadata: testnod.TestRunMetadata{
				Branch:    config.Branch,
				CommitSHA: config.CommitSHA,
				RunURL:    config.RunURL,
				BuildID:   config.BuildID,
			},
		},
	}

	serverResponse, err := testnod.CreateTestRun(config.UploadURL, config.Token, uploadRequest)
	if err != nil {
		fmt.Printf("Error creating test run on TestNod: %v\n", err)
		exitBasedOnIgnoreFailures(config.IgnoreFailures)
	}

	fmt.Println("Created test run, uploading JUnit XML file...")
	err = upload.UploadJUnitXmlFile(config.FilePath, serverResponse.PresignedURL)

	if err != nil {
		fmt.Println("There was an error uploading the file to TestNod. We've been notified and will look into it. Sorry for the inconvenience.")
		exitBasedOnIgnoreFailures(config.IgnoreFailures)
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
