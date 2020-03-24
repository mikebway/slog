package cmd

// Unit tests for the Cobra command line parsers

import (
	"bytes"
	"testing"
	"time"

	"github.com/mikebway/slog/s3"
	"github.com/stretchr/testify/require"
)

// Initialization block
func init() {
	// When running unit test on the command line parser, signal that the actual operations
	// should not be executed, only the parsing.
	unitTesting = true
}

// executeCommand invokes Execute() while capturing its output
// to return for analysis. Any error will have been collected in the
// executeError package global.
func executeCommand(args ...string) string {

	// Ensure that we are in a sweet and innocent state
	resetCommand()

	// Arrange to collect the output in a buffer
	buf := new(bytes.Buffer)
	rootCmd.SetOutput(buf)

	// Set the arguments and invoke the normal Execute() package entry point
	rootCmd.SetArgs(args)
	Execute()

	// Return the output as a string
	return buf.String()
}

// resetCommand clears both command specific parameter values and
// global ones so that tests can be run in a known "virgin" state.
func resetCommand() {

	// Reset read command specific values
	startDateStr = ""
	startDateTime = time.Time{}
	windowStr = ""
	window = time.Duration(0)
	contentTypeStr = ""
	slogSession = nil

	// Reset the global values
	executeError = nil
	region = ""
	path = ""

	// Clear and then re-initialize all the flags definitions
	rootCmd.ResetFlags()
	readCmd.ResetFlags()
	initRootFlags()
	initReadFlags()
}

// TestExecute maximizes coverage by invoking cmd.Execute().
// We get less information back from cmd.Execute() so don't invoke it for the
// majority of our tests, going around it for them.
func TestExecute(t *testing.T) {

	// Execute the slog command with no parameters
	output := executeCommand()

	// We should have a subcommand required command and a complete usage dump
	require.NotNil(t, executeError, "there should have been an error")
	require.Equal(t, "subcommand is required", executeError.Error(), "Expected subcommand required error")
	require.Contains(t, output,
		"Slog is a CLI utility for reading and culling web access logs stored in S3",
		"Expected full usage display")
}

// TestBareCommand examines the case where no parameters are provided
func TestBareCommand(t *testing.T) {

	// Run a blank command
	output := executeCommand()

	// We should have a subcommand required command and a complete usage dump
	require.NotNil(t, executeError, "there should have been an error")
	require.Equal(t, "subcommand is required", executeError.Error(), "Expected subcommand required error")
	require.Contains(t, output,
		"Slog is a CLI utility for reading and culling web access logs stored in S3",
		"Expected full usage display")
}

// TestBareReadCommand examines the case where a read command is requested
// but no parameters are provided
func TestBareReadCommand(t *testing.T) {

	// Run the command
	output := executeCommand("read")

	// We should have a bucket required error but no usage displayed
	require.NotNil(t, executeError, "there should have been an error")
	require.Equal(t, "An S3 bucket name must be provided", executeError.Error(), "Expected S3 bucket name required error")
	require.Empty(t, output, "Expected no usage display")
}

// TestMinimumReadCommand provides only the required parmeters and confirms that
// the parser is happy and assumes the exptected default values.
func TestMinimumReadCommand(t *testing.T) {

	// The following should parse happilly
	executeCommand("read", "my-bucket")
	require.Nil(t, executeError, "error seen parsing minimum read command line")
	require.Equal(t, "us-east-1", slogSession.Region, "Default region set incorrectly: %s", region)
	require.Equal(t, "root", slogSession.Folder, "Default path set incorrectly: %s", path)
	require.Equal(t, s3.BASIC, slogSession.Content, "Default content type set incorrectly: %s", path)
	expectedStartDateTime, _ := time.Parse(time.RFC3339, "2020-01-01T00:00:00+00:00")
	require.Equal(t, expectedStartDateTime, slogSession.StartDateTime, "Default start date time set incorrectly: %v", startDateTime)
	expectedEndDateTime := expectedStartDateTime.Add(time.Hour)
	require.Equal(t, expectedEndDateTime, slogSession.EndDateTime, "Default winwow set incorrectly: %v", window)
}

// TestReadCommandTooMany examines the case where a read command is requested
// with too many non-flag parameters.
func TestReadCommandTooMany(t *testing.T) {

	// Run the command
	output := executeCommand("read", "bucket", "one-too-many")

	// We should have a only one bucket name expected error and no usage display
	require.NotNil(t, executeError, "there should have been an error")
	require.Equal(t, "Only expected a single bucket name argument", executeError.Error(), "Expected S3 bucket name required error")
	require.Empty(t, output, "Expected no usage display")
}

// TestReadCommandBadStart examines the case where a read command is requested
// with an invalid start time
func TestReadCommandBadStart(t *testing.T) {

	// Run the command
	output := executeCommand("read", "bucket", "--start", "blargle")

	// We should have am invalid start time error and no usage display
	require.NotNil(t, executeError, "there should have been an error")
	require.Equal(t,
		"Invalid start date time: parsing time \"blargle\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"blargle\" as \"2006\"",
		executeError.Error(), "Expected invalid --start value error")
	require.Empty(t, output, "Expected no usage display")
}

// TestReadCommandStart examines whether start time parsing is correctly handled by the read command
func TestReadCommandStart(t *testing.T) {

	// Run the command
	executeCommand("read", "bucket", "--start", "2020-03-04T05:06:07+08:00")

	// We should have a subcommand required command and a complete usage dump
	require.Nil(t, executeError, "error seen parsing valid start time")
	require.Equal(t, 2020, startDateTime.Year(), "Expected start year did not match")
	require.Equal(t, time.March, startDateTime.Month(), "Expected start month did not match")
	require.Equal(t, 4, startDateTime.Day(), "Expected start day did not match")
	require.Equal(t, 5, startDateTime.Hour(), "Expected start hour did not match")
	require.Equal(t, 6, startDateTime.Minute(), "Expected start minute did not match")
	require.Equal(t, 7, startDateTime.Second(), "Expected start second did not match")
	require.Equal(t, 0, startDateTime.Nanosecond(), "Expected start nanosecond did not match")
	_, offset := startDateTime.Zone()
	require.Equal(t, 8*3600, offset, "Expected start time zone did not match")
}

// TestReadCommandBadWindow examines the case where a read command is requested
// with an invalid time window
func TestReadCommandBadWindow(t *testing.T) {

	// Run the command
	output := executeCommand("read", "bucket", "--window", "blargle")

	// We should have am invalid time window error and no usage display
	require.NotNil(t, executeError, "there should have been an error")
	require.Equal(t,
		"Invalid time window: Cannot parse time window length",
		executeError.Error(), "Expected invalid --window value error")
	require.Empty(t, output, "Expected no usage display")
}

// TestReadCommandWindow examines whether time window parsing is correctly handled by the read command
func TestReadCommandWindow(t *testing.T) {

	// Run the command with a window in days and check the result
	executeCommand("read", "bucket", "--window", "7d")
	require.Nil(t, executeError, "error seen parsing valid window time of 7 days")
	require.Equal(t, 168.0, window.Hours(), "Expected 7 day window did not match")

	// Run the command with a window in hours and check the result
	executeCommand("read", "bucket", "--window", "12h")
	require.Nil(t, executeError, "error seen parsing valid window time of 12 hours")
	require.Equal(t, 12.0, window.Hours(), "Expected 12 hour window did not match")

	// Run the command with a window in days and check the result
	executeCommand("read", "bucket", "--window", "25m")
	require.Nil(t, executeError, "error seen parsing valid window time of 25 minutes")
	require.Equal(t, 25.0, window.Minutes(), "Expected 25 minute window did not match")

	// Run the command with a window in days and check the result
	executeCommand("read", "bucket", "--window", "95s")
	require.Nil(t, executeError, "error seen parsing valid window time of 95 seconds")
	require.Equal(t, 95.0, window.Seconds(), "Expected 95 second window did not match")
}

// TestReadCommandBadContentType checks that an invalid content type
// flag value is rejected with and error
func TestReadCommandBadContentType(t *testing.T) {

	// Run the command with an invalid content type
	executeCommand("read", "bucket", "--content", "cheese")
	require.NotNil(t, executeError, "there should have been an error")
	require.Contains(t, executeError.Error(), "cheese", "error decription did not contain the bad content type")
}

// TestReadCommandContentTypes checks that all of the valid content types are accepted
func TestReadCommandContentTypes(t *testing.T) {

	// Run the command specifying the basic content type
	executeCommand("read", "bucket", "--content", "basic")
	require.Nil(t, executeError, "basic should have been an acceptable content type")
	require.Equal(t, s3.BASIC, slogSession.Content, "SlogSession not populated with the right content type")

	// Run the command specifying the request content type
	executeCommand("read", "bucket", "--content", "requestid")
	require.Nil(t, executeError, "request should have been an acceptable content type")
	require.Equal(t, s3.REQUESTID, slogSession.Content, "SlogSession not populated with the right content type")

	// Run the command specifying the bucket content type
	executeCommand("read", "bucket", "--content", "bucket")
	require.Nil(t, executeError, "bucket should have been an acceptable content type")
	require.Equal(t, s3.BUCKET, slogSession.Content, "SlogSession not populated with the right content type")

	// Run the command specifying the rich content type
	executeCommand("read", "bucket", "--content", "rich")
	require.Nil(t, executeError, "rich should have been an acceptable content type")
	require.Equal(t, s3.RICH, slogSession.Content, "SlogSession not populated with the right content type")

	// Run the command specifying the raw content type
	executeCommand("read", "bucket", "--content", "raw")
	require.Nil(t, executeError, "raw should have been an acceptable content type")
	require.Equal(t, s3.RAW, slogSession.Content, "SlogSession not populated with the right content type")
}
