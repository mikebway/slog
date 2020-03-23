package s3

// Unit tests for the slogs S3 read functions

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestReadEndToEnd runs the full, happy path, pipeline of the read command.
func TestReadEndToEnd(t *testing.T) {

	// Will substitute our own pipe for stdout to collect the log output
	// but must be carefule to always restore stadt and close the pripe files.
	originalStdout := os.Stdout
	readFile, writeFile, err := os.Pipe()
	defer func() {
		// Restore original stdout if something goes wrong
		os.Stdout = originalStdout
		writeFile.Close()
		readFile.Close()
	}()

	// Set our own pipe as stdout
	assert.Nil(t, err, "Failed to create pipe for stdout: %v", err)
	os.Stdout = writeFile

	// Obtain a session (inactive) populated with target bucket values
	slogSess := newTestSlogSession()

	// Run the pipeline, collecting the log output in our writeFile
	err = DisplayLog(slogSess)

	// Restore stdout and close the write end of the pipe so that we can see how the test goes!!
	os.Stdout = originalStdout
	writeFile.Close()

	// Confirm that DisplayLog did not return an error
	assert.Nil(t, err, "DisplayLog failed unexpectedly: %v", err)

	// Leats see what the log wrote ...
	outputBytes, err := ioutil.ReadAll(readFile)
	output := string(outputBytes)
	assert.Contains(t, output, targetContains, "Log output did not contain the expected data")
}

// TestReadBadBucket examines what happens if the specified. It should fail fast.
func TestReadBadBucket(t *testing.T) {

	// Build a session object with an invalid bucktt name
	slogSess := newTestSlogSession()
	slogSess.Bucket = "there-is-no-bucket-with-this-name-xyz123"

	// Try to display the logs form the non-existent bucket
	err := DisplayLog(slogSess)

	// If that did not return an error I will eat my hat!
	assert.NotNil(t, err, "Should not have been able to display logs from a non-existent bucket")
}

// TestReadSessiontFailure looks at how DisplayLog handles a failure to activate the
// SlogSession with AWS session and S3 client handles.
func TestReadSessiontFailure(t *testing.T) {

	// Trick AWS session.NewSession into failing by setting an invalid environmentt variable
	const envVarName = "AWS_S3_USE_ARN_REGION"
	originalEnvVarValue := os.Getenv(envVarName)
	defer func() {
		os.Setenv(envVarName, originalEnvVarValue)
	}()
	os.Setenv(envVarName, "this-should-fail")

	// Try to display the logs and confirm that it blows up
	slogSess := newTestSlogSession()
	err := DisplayLog(slogSess)
	assert.NotNil(t, err, "Should not have been able to display logs with a session activation error")
}
