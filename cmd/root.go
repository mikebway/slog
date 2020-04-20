/*
Package cmd implements command line parsing via the Cobra and Viper modules for the slog CLI utility.
The slog utility manages web access logs stored in S3.
*/
package cmd

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	unitTesting  = false // Set to true when running unit tests
	executeError error   // The error value obtained by Execute(), captured for unit test purposes
	region       string  // The AWS regon to target
	path         string  // the log folder path within the S3 bucket
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "slog",
	Short: "The slog utility manages web access logs stored in S3",
	Long: `Slog is a CLI utility for reading and culling web access logs stored in S3.

Typically, the logs managed are those generated in response to access to static web assets
themselves served directly from S3.`,

	SilenceUsage:  true, // Only display help when explicitly requested
	SilenceErrors: true, // Only display errors once
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if executeError = rootCmd.Execute(); executeError != nil {
		fmt.Println(executeError)
		if !unitTesting {
			os.Exit(1)
		}
	}
}

// Load time initialization - called automatically
func init() {

	// Initialize the flags that apply to the root command and, potentially, to subcommands
	initRootFlags()
}

// initRootFlags is called from init() to define the flags that apply to the root
// command, and might be inherited by its subcommands. It is defined separately from
// init() so that it can be invoked by unit tests when they need to reset the playing field.
func initRootFlags() {

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&region, "region", "us-east-1", "the aws region to target")
	rootCmd.PersistentFlags().StringVar(&path, "path", "root", `The path of the log data within the S3 bucket`)

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//  rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// ============================================================================
// The following ar provided to support unit tests. In particular, they allow
// the tests for the main package to ensure that the environment is reset
// from any prior tests.
// ============================================================================

// PrepForExecute is FOR UNIT TESTING ONLY. It is is intended to be invoked by
// the main package's tests to clear any previous test debris and capture output
// in the returned buffer. It accepts a variable number of string values to
// be set as arguments for command execution.
func PrepForExecute(args ...string) *bytes.Buffer {

	// Have our string slice accepting sibling do all the work
	return prepForExecute(args)
}

// prepForExecute is a package scoped function accepting a string array of
// command line argument values. It resets the session environment and
// establishs a buffer to capture the output of the subsequent command
// execution.
func prepForExecute(args []string) *bytes.Buffer {

	// Ensure that we are in a sweet and innocent state
	resetCommand()

	// Set the arguments and invoke the normal Execute() package entry point
	rootCmd.SetArgs(args)

	// Arrange to collect the output in a buffer
	buf := new(bytes.Buffer)
	rootCmd.SetOutput(buf)

	// Return the, as yet, empty buffer
	return buf
}

// resetCommand clears both command specific parameter values and
// global ones so that tests can be run in a known "virgin" state.
func resetCommand() {

	// Tell the world that we are unit testing and not to get
	// too carried away
	unitTesting = true

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
