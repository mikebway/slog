package s3

// Unit tests for the slogs S3 core functions

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Variables required to drive the tests
var (

	// A valid AWS S3 web log bucket and path with usable log data must be defined
	targetRegion   string
	targetBucket   string
	targetFolder   string
	targetContains string

	// A time window for which there are multiple log objects must be define
	targetStartDateTime time.Time
	targetEndDateTime   time.Time
)

// Initialization block
func init() {

	// We are really doing integration tests as mush as unit tests! Load the
	// target AWS environment information frpm environment variables.
	targetRegion = os.Getenv("SLOG_TEST_REGION")
	targetBucket = os.Getenv("SLOG_TEST_BUCKET")
	targetFolder = os.Getenv("SLOG_TEST_FOLDER")
	targetContains = os.Getenv("SLOG_TEST_CONTAINS")
	targetStartTimeStr := os.Getenv("SLOG_TEST_START_DATETIME")
	targetEndTimeStr := os.Getenv("SLOG_TEST_END_DATETIME")

	// If any of the required environment variables are mossing or
	// valid, barf and tell the user.
	isEnvValid := true
	if len(targetRegion) == 0 ||
		len(targetBucket) == 0 ||
		len(targetFolder) == 0 ||
		len(targetStartTimeStr) == 0 ||
		len(targetEndTimeStr) == 0 ||
		len(targetContains) == 0 {
		isEnvValid = false
		fmt.Println("ERROR: one or more of the required test environment variables is missing")
	}

	// If we are good so far, parse the time values
	if isEnvValid {
		var err error
		targetStartDateTime, err = time.Parse(time.RFC3339, targetStartTimeStr)
		isTimeValid := err == nil
		targetEndDateTime, err = time.Parse(time.RFC3339, targetEndTimeStr)
		isTimeValid = isTimeValid && err == nil
		if !isTimeValid {
			isEnvValid = false
			fmt.Println("ERROR: one or more of the required test time environment variables is invalid")
		}
	}

	// If the environment variable parsing failed then tell the user what they should look like
	// and abort the test run
	if !isEnvValid {
		fmt.Println(`
To run the slog S3 package tests, all of the following environment variables should
be set, pointing to a real AWS S3 log bucket with a time window that covers multiple
log objects / seconds of data and a smaple that will be found contained within that
log data. For example:

export SLOG_TEST_REGION=us-east-1
export SLOG_TEST_BUCKET=log.mikebroadway.com
export SLOG_TEST_FOLDER=root
export SLOG_TEST_START_DATETIME=2020-03-20T13:30:00Z
export SLOG_TEST_END_DATETIME=2020-03-20T14:00:00Z
export SLOG_TEST_CONTAINS="AA960FCC76F5673E WEBSITE.GET.OBJECT robots.txt"`)
		os.Exit(1)
	}
}

// newTestSlogSession creates a SlogSession populated with the test target values
func newTestSlogSession() *SlogSession {
	return &SlogSession{
		Region:        targetRegion,
		Bucket:        targetBucket,
		Folder:        targetFolder,
		StartDateTime: targetStartDateTime,
		EndDateTime:   targetEndDateTime,
	}
}

// TestActivateSessiont confirms that the activateSession happy path populates a SlogSession
// structure with both an AWS session and an S3 client.
func TestActivateSessiont(t *testing.T) {

	// Create and activate the session
	slogSess := newTestSlogSession()
	err := activateSession(slogSess)
	assert.True(t, err == nil, "activateSession should have succeeded: %v", err)

	// If we have a healthy session, all be it largely unpopulated ...
	if err == nil {

		// While we have a populated session, see what happens when we ask to activate it
		// a second time - it should return without making any changes

		// Make a not of the AWS values as they are now in the session
		awsSession := slogSess.awsSession
		s3 := slogSess.s3

		// Activating it for a second time
		err = activateSession(slogSess)
		assert.True(t, err == nil, "activateSession twice should have succeeded: %v", err)
		assert.Equal(t, awsSession, slogSess.awsSession, "Double activation should not have changed the AWS session")
		assert.Equal(t, s3, slogSess.s3, "Double activation should not have changed the S3 client")
	}
}

// TestActivateSessiontFailure confirms that the activateSession returns an error if
// a region is not supplied.
func TestActivateSessiontFailure(t *testing.T) {

	// Trick AWS session.NewSession into failing by setting an invalid environmentt variable
	const envVarName = "AWS_S3_USE_ARN_REGION"
	originalEnvVarValue := os.Getenv(envVarName)
	defer func() {
		os.Setenv(envVarName, originalEnvVarValue)
	}()
	os.Setenv(envVarName, "this-should-fail")

	// Create and activate the session
	slogSess := newTestSlogSession()
	err := activateSession(slogSess)
	assert.True(t, err != nil, "activateSession should have failed with a fad environment")
}
