package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

const (

	// Start time format
	layoutStartTime = "2006-01-02-15-04-05-0700"
)

var (
	path         string        // the log folder path within the S3 bucket
	startDateStr string        // flag value defining the start time of the window to be processed
	startDate    time.Time     // the start time of the window to be processed
	windowStr    string        // flag value defining the duration / time span to be considered
	window       time.Duration // the duration / time span to be considered
)

// readCmd represents the read command
var readCmd = &cobra.Command{
	Use:   "read bucket",
	Short: "Display S3 hosted web logs for a given time window",
	Long: `Given a start date and time, together with a time window, displays the
S3 hosted web logs from a specified bucket for that time window.`,

	RunE: func(cmd *cobra.Command, args []string) error {

		// There must be an S3 bucket name
		if len(args) == 0 {
			return errors.New("An S3 bucket name must be provided")
		}
		if len(args) > 1 {
			return errors.New("Only expected a single bucket name argument")
		}

		// Parse the start time
		var err error
		startDate, err = time.Parse(layoutStartTime, startDateStr)
		if err != nil {
			return fmt.Errorf("Invalid start date time: %w", err)
		}

		// Parse the time window
		window, err = parseTimeWindow(windowStr)
		if err != nil {
			return fmt.Errorf("Invalid time window: %w", err)
		}

		// All is well with the command formating (to the best of our present knowledge).
		// Go ahead and do the work unless we are unit testing.
		fmt.Printf("Reading logs from %v/%v for with start=%v, window=%v seconds",
			args[0], path, startDate.Format(layoutStartTime), window.Seconds())
		if !unitTesting {
			fmt.Println("Read command code will be called here")
		}

		// Command line parsing succeeded even if the execution failed
		return nil
	},
}

func init() {
	rootCmd.AddCommand(readCmd)

	// Here you will define your flags and configuration settings.

	// Local flag definitions
	readCmd.Flags().StringVar(&startDateStr, "start", "2020-01-01-00-00-00-0000",
		`Start date time in the form YYYY-MM-DD-hh-mm-ss-0000 form with time zone offset`)
	readCmd.Flags().StringVar(&windowStr, "window", "1h",
		`Time window in the days (d), hours (h), minutes (m) or seconds (s).
For example '90s' for 90 seconds. '36h' for 36 hours.`)
	readCmd.Flags().StringVar(&path, "path", "root",
		`The path of the log data within the S3 bucket`)
}

// Parse a time window string into a duration
func parseTimeWindow(wstr string) (time.Duration, error) {

	// The string must be at least two characters in length
	l := len(wstr)
	if l > 1 {

		// The last character tells us the type of the number that preceeds it (hours, minites, etc)
		// The characters before the type should be an integer count
		i, err := strconv.Atoi(wstr[0 : l-1])
		if err == nil {

			// Switch on the type to calucalte the appropriate duration
			switch wstr[l-1:] {

			case "d":
				return time.Hour * time.Duration(i*24), nil
			case "h":
				return time.Hour * time.Duration(i), nil
			case "m":
				return time.Minute * time.Duration(i), nil
			case "s":
				return time.Second * time.Duration(i), nil
			}
		}
	}

	// The window string is invalid
	return 0, errors.New("Cannot parse time window length")
}
