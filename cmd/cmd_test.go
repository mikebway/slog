package cmd

import (
	"bytes"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// Unit tests for the Cobra command line parsers

// Initialization block
func init() {
	// When running unit test on the command line parser, signal that the actual operations
	// should not be executed, only the parsing.
	unitTesting = true
}

// Modified from https://chromium.googlesource.com/external/github.com/spf13/cobra/+/refs/heads/master/command_test.go
func executeCommand(args ...string) (output string, err error) {
	_, output, err = executeCommandC(args...)
	return output, err
}

// Modified from https://chromium.googlesource.com/external/github.com/spf13/cobra/+/refs/heads/master/command_test.go
func executeCommandC(args ...string) (c *cobra.Command, output string, err error) {

	// Handle the special case of no arguaments. Cobra treats nil arguements
	// differently from empty arguments, typing itself in a small knot and inventing
	// something from the stack.
	if args == nil {
		args = []string{}
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOutput(buf)
	rootCmd.SetArgs(args)
	c, err = rootCmd.ExecuteC()
	return c, buf.String(), err
}

// TestExecute maximizes coverage by invoking cmd.Execute().
// We get less information back from cmd.Execute() so don't invoke it for the
// majority of our tests, going around it for them.
func TestExecute(t *testing.T) {

	// Execute the slog command with no parameters
	buf := new(bytes.Buffer)
	rootCmd.SetOutput(buf)
	rootCmd.SetArgs([]string{})
	Execute()

	// We should have a subcommand required command and a complete usage dump
	assert.Equal(t, "subcommand is required", executeError.Error(), "Expected subcommand required error")
	assert.Contains(t, buf.String(),
		"Slog is a CLI utility for reading and culling web access logs stored in S3",
		"Expected full usage display")
}

// TestBareCommand examines the case where no parameters are provided
func TestBareCommand(t *testing.T) {

	// Run a blank command
	output, err := executeCommand()

	// We should have a subcommand required command and a complete usage dump
	assert.Equal(t, "subcommand is required", err.Error(), "Expected subcommand required error")
	assert.Contains(t, output,
		"Slog is a CLI utility for reading and culling web access logs stored in S3",
		"Expected full usage display")
}

// TestBareReadCommand examines the case where a read command is requested
// but no parameters are provided
func TestBareReadCommand(t *testing.T) {

	// Run the command
	output, err := executeCommand("read")

	// We should have a subcommand required command and a complete usage dump
	assert.Equal(t, "An S3 bucket name must be provided", err.Error(), "Expected S3 bucket name required error")
	assert.Contains(t, output, "slog read bucket [flags]", "Expected read command usage display")
}

// TestReadCommandTooMany examines the case where a read command is requested
// with too many non-flag parameters.
func TestReadCommandTooMany(t *testing.T) {

	// Run the command
	output, err := executeCommand("read", "bucket", "one-too-many")

	// We should have a subcommand required command and a complete usage dump
	assert.Equal(t, "Only expected a single bucket name argument", err.Error(), "Expected S3 bucket name required error")
	assert.Contains(t, output, "slog read bucket [flags]", "Expected read command usage display")
}

// TestReadCommandBadStart examines the case where a read command is requested
// with an invalid start time
func TestReadCommandBadStart(t *testing.T) {

	// Run the command
	output, err := executeCommand("read", "bucket", "--start", "blargle")

	// We should have a subcommand required command and a complete usage dump
	assert.Equal(t,
		"Invalid start date time: parsing time \"blargle\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"blargle\" as \"2006\"",
		err.Error(),
		"Expected invalid --start value error")
	assert.Contains(t, output, "slog read bucket [flags]", "Expected read command usage display")
}

// TestReadCommandStart examines whether start time parsing is correctly handled by the read command
func TestReadCommandStart(t *testing.T) {

	// Run the command
	_, err := executeCommand("read", "bucket", "--start", "2020-03-04T05:06:07+08:00")

	// We should have a subcommand required command and a complete usage dump
	assert.Nil(t, err, "error seen parsing valid start time")
	assert.Equal(t, 2020, startDateTime.Year(), "Expected start year did not match")
	assert.Equal(t, time.March, startDateTime.Month(), "Expected start month did not match")
	assert.Equal(t, 4, startDateTime.Day(), "Expected start day did not match")
	assert.Equal(t, 5, startDateTime.Hour(), "Expected start hour did not match")
	assert.Equal(t, 6, startDateTime.Minute(), "Expected start minute did not match")
	assert.Equal(t, 7, startDateTime.Second(), "Expected start second did not match")
	assert.Equal(t, 0, startDateTime.Nanosecond(), "Expected start nanosecond did not match")
	_, offset := startDateTime.Zone()
	assert.Equal(t, 8*3600, offset, "Expected start time zone did not match")
}

// TestReadCommandBadWindow examines the case where a read command is requested
// with an invalid time window
func TestReadCommandBadWindow(t *testing.T) {

	// Run the command
	output, err := executeCommand("read", "bucket", "--window", "blargle")

	// We should have a subcommand required command and a complete usage dump
	assert.Equal(t,
		"Invalid time window: Cannot parse time window length",
		err.Error(),
		"Expected invalid --window value error")
	assert.Contains(t, output, "slog read bucket [flags]", "Expected read command usage display")
}

// TestReadCommandWindow examines whether time window parsing is correctly handled by the read command
func TestReadCommandWindow(t *testing.T) {

	// Run the command with a window in days and check the result
	var err error
	_, err = executeCommand("read", "bucket", "--window", "7d")
	assert.Nil(t, err, "error seen parsing valid window time of 7 days")
	assert.Equal(t, 168.0, window.Hours(), "Expected 7 day window did not match")

	// Run the command with a window in hours and check the result
	_, err = executeCommand("read", "bucket", "--window", "12h")
	assert.Nil(t, err, "error seen parsing valid window time of 12 hours")
	assert.Equal(t, 12.0, window.Hours(), "Expected 12 hour window did not match")

	// Run the command with a window in days and check the result
	_, err = executeCommand("read", "bucket", "--window", "25m")
	assert.Nil(t, err, "error seen parsing valid window time of 25 minutes")
	assert.Equal(t, 25.0, window.Minutes(), "Expected 25 minute window did not match")

	// Run the command with a window in days and check the result
	_, err = executeCommand("read", "bucket", "--window", "95s")
	assert.Nil(t, err, "error seen parsing valid window time of 95 seconds")
	assert.Equal(t, 95.0, window.Seconds(), "Expected 95 second window did not match")
}
