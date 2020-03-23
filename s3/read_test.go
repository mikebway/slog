package s3

// Unit tests for the slogs S3 read functions

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
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

// TestMissingLogObject sees how fetchLogObjectData(..) handles an error when downloading
// log data from a given key.
func TestMissingLogObject(t *testing.T) {

	// Obtain an activated session
	slogSess := newTestSlogSession()
	err := activateSession(slogSess)
	assert.Nil(t, err, "activateSession should have succeeded: %v", err)

	// Establish the channels needed to communicate with TestMissingLogObject(..) as
	// a Go routine (though we will not run it as a Go routine)
	errChan := make(chan error, 5)               // Used to signal errors that require the app DisplayLog to terminate
	keyChan := make(chan string, 5)              // Distributes S3 object keys listed from the log bucket
	dataChan := make(chan *aws.WriteAtBuffer, 5) // Distributes AWS wrapped byte buffers downloaded from S3 objects

	// Whatever happens with this test, we should not leave any channels open
	defer func() {
		close(errChan)
		close(keyChan)
	}()

	// Load a key value intto the keyChan that we know will not exist in the bucket.
	// keyChan is buffered so will not halt waiting for somebody to read from it
	keyChan <- "I-do-not-exist-2300-12-31"

	// The function we are testing should fail quickly so there is no need to spin it
	// up as a Go routine in its own thread. We log what we are doing to help a little
	// if the human observer needs to diagnose where a test timeout occurred.
	fmt.Println("Launching fetchLogObjectData(..) to see it fail")
	go fetchLogObjectData(slogSess, keyChan, dataChan, errChan)

	// We should arrive back here long before the test harness times us out
	fmt.Println("fetchLogObjectData(..) returned, now fetching the expected error")
	err = <-errChan
	assert.NotNil(t, err, "fetchLogObjectData should have piped an error: %v", err)

	// dataChan should have been closed but the only way to find out if that is the case
	// is to try to read from it and hope the test does not time out wiating on it
	fmt.Println("Confirming that the data channel has been closed")
	<-dataChan
}
