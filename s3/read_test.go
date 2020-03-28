package s3

// Unit tests for the slogs S3 read functions

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/require"
)

// captureLog wraps DisplayLog(..) to capture the output for subsequent examination
// by a test.
func captureLog(slogSess *SlogSession) (string, error) {

	// We substitute our own pipe for stdout to collect the log output
	// but must be carefule to always restore stadt and close the pripe files.
	originalStdout := os.Stdout
	readFile, writeFile, err := os.Pipe()
	if err != nil {
		return "", fmt.Errorf("Failed to create pipe for stdout: %v", err)
	}

	// Restore original stdout even if something goes wrong
	defer func() {
		os.Stdout = originalStdout
		writeFile.Close()
		readFile.Close()
	}()

	// Set our own pipe as stdout
	os.Stdout = writeFile

	// Run the pipeline, collecting the log output in our writeFile
	err = DisplayLog(slogSess)
	if err != nil {
		return "", err
	}

	// Restore stdout and close the write end of the pipe so that we can collect the ouput
	os.Stdout = originalStdout
	writeFile.Close()

	// Gather the output into a byte buffer
	outputBytes, err := ioutil.ReadAll(readFile)
	if err != nil {
		return "", fmt.Errorf("Failed to read pipe for stdout: %v", err)
	}

	// Return the output as a string
	return string(outputBytes), nil
}

// TestReadEndToEnd runs the full, happy path, pipeline of the read command.
func TestReadEndToEnd(t *testing.T) {

	// Obtain a session (inactive) populated with target bucket values
	// but ask for raw content to get the most data to match with our target string below
	slogSess := newTestSlogSession()
	slogSess.Content = RAW

	// Run the DisplayLog(..) pipeline, collecting the log output for analysis
	output, err := captureLog(slogSess)

	// Confirm that DisplayLog did not return an error
	require.Nil(t, err, "DisplayLog or capture failed unexpectedly: %v", err)

	// Check that the log conatianed what we expected
	require.Contains(t, output, targetContains, "Log output did not contain the expected data")
}

// TestReadBadBucket examines what happens if the specified bucket does not exist.
// It should fail fast!
func TestReadBadBucket(t *testing.T) {

	// Build a session object with an invalid bucktt name
	slogSess := newTestSlogSession()
	slogSess.LogBucket = "there-is-no-bucket-with-this-name-xyz123"

	// Try to display the logs form the non-existent bucket
	err := DisplayLog(slogSess)

	// If that did not return an error I will eat my hat!
	require.NotNil(t, err, "Should not have been able to display logs from a non-existent bucket")
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
	require.NotNil(t, err, "Should not have been able to display logs with a session activation error")
}

// TestMissingLogObject sees how fetchLogObjectData(..) handles an error when downloading
// log data from a given key.
func TestMissingLogObject(t *testing.T) {

	// Obtain an activated session
	slogSess := newTestSlogSession()
	err := activateSession(slogSess)
	require.Nil(t, err, "activateSession should have succeeded: %v", err)

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
	require.NotNil(t, err, "fetchLogObjectData should have piped an error: %v", err)

	// dataChan should have been closed but the only way to find out if that is the case
	// is to try to read from it and hope the test does not time out wiating on it
	fmt.Println("Confirming that the data channel has been closed")
	<-dataChan
}

// TestReadBadContentType examines what happens if the specified. It should fail fast.
func TestReadBadContentType(t *testing.T) {

	// Build a session object with an invalid content type
	slogSess := newTestSlogSession()
	slogSess.Content = RAW + 197 // This is not a valid content type

	// Try to display the logs form the non-existent bucket
	err := DisplayLog(slogSess)

	// If that did not return an error I will eat my hat!
	require.NotNil(t, err, "Should not have been able to display logs with an invalid content type")
}

// TestReadContentTypes goes some way to confirmin that all of the different content types
// work. Short of a well trained AI model, there is no way to easily confirm that the content
// contain exactly the right fields - we will leave that for a manuall inspection - so all
// we do here is confirm that each type produces a different and non-zero answer.
func TestReadContentTypes(t *testing.T) {

	// Start with a default session definition
	slogSess := newTestSlogSession()

	// Basic content will be the smallest
	slogSess.Content = BASIC
	basicOutput, err := captureLog(slogSess)
	require.Nil(t, err, "Failed to capture basic log content: %v", err)
	basicLength := len(basicOutput)

	// Repeat for Request ID content
	slogSess.Content = REQUESTID
	requestIDOutput, err := captureLog(slogSess)
	require.Nil(t, err, "Failed to capture request ID log content: %v", err)
	requestIDLength := len(requestIDOutput)

	// Repeat for bucket content
	slogSess.Content = BUCKET
	bucketOutput, err := captureLog(slogSess)
	require.Nil(t, err, "Failed to capture bucket log content: %v", err)
	bucketLength := len(bucketOutput)

	// Repeat for rich content
	slogSess.Content = RICH
	richOutput, err := captureLog(slogSess)
	require.Nil(t, err, "Failed to capture bucket log content: %v", err)
	richLength := len(richOutput)

	// Repeat for rich content
	slogSess.Content = RAW
	rawOutput, err := captureLog(slogSess)
	require.Nil(t, err, "Failed to capture bucket log content: %v", err)
	rawLength := len(rawOutput)

	// Compare the length to ensure that they are as we expect, relatively speaking at least
	require.Greater(t, basicLength, 0, "Basic content length must be longer than zero bytes")
	require.Greater(t, rawLength, basicLength, "Raw content length must be longer than basic")
	require.Greater(t, rawLength, requestIDLength, "Raw content length must be longer than request ID")
	require.Greater(t, rawLength, bucketLength, "Raw content length must be longer than bucket")
	require.Greater(t, rawLength, richLength, "Raw content length must be longer than rich")
	require.Greater(t, requestIDLength, basicLength, "Raw content length must be longer than basic")
	require.Greater(t, bucketLength, basicLength, "Bucket content length must be longer than basic")
	require.Greater(t, richLength, bucketLength, "Bucket content length must be longer than bucket")
}

// TestReadFilter looks at whether content can be filtered by specifying a source bucket.
// The test is limited in that it does not analyze the returned content in depth, only
// confirm that filtering reduces content. True testing of this feature must be left
// to manual inspection.
func TestReadFilter(t *testing.T) {

	// Caputure the output of an unfiltered run but asking for the source bucket names
	// to be included so that we can find out what they might be regarldess of who
	// configured tests.
	slogSess := newTestSlogSession()
	slogSess.Content = BUCKET
	output, err := captureLog(slogSess)
	require.Nil(t, err, "Error capturing initial log content with bucket name: %v", err)
	originalLenWithBucketName := len(output)
	require.Greater(t, originalLenWithBucketName, 0, "No content captured in initial request for content with source bucket name: %v", err)

	// Extract a bucket name from the first line of content - it is the first field
	breakIndex := strings.IndexByte(output, ' ')
	knownSourceBucket := output[0:breakIndex]
	require.Greater(t, len(knownSourceBucket), 0, "Failed to extract source bucket name from initial content")

	// Now ask for basic content filtered to only include that for the bucket name we just found
	slogSess.Content = BASIC
	slogSess.SourceBuckets = make([]string, 1)
	slogSess.SourceBuckets[0] = knownSourceBucket
	output, err = captureLog(slogSess)
	require.Nil(t, err, "Error capturing log content filtered for bucket name %s: %v", knownSourceBucket, err)
	lenFilteredForKnownBucket := len(output)
	require.Greater(t, lenFilteredForKnownBucket, 0, "No content captured filtering for known source bucket %s: %v", knownSourceBucket, err)
	require.Less(t, lenFilteredForKnownBucket, originalLenWithBucketName, "Obtained longer content when filtering than unfiltered with bucket names!")

	// Finally, repeat filtering for a name we are pretty certain will not be there
	slogSess.SourceBuckets[0] = "not a valid bucket name !?&%$#,|"
	output, err = captureLog(slogSess)
	require.Nil(t, err, "Error capturing log content filtered for invalid bucket name %s: %v", slogSess.SourceBuckets[0], err)
	require.Equal(t, len(output), 0, "Should have had no content filtering for invalid bucket name %s: %v", knownSourceBucket, err)
}
