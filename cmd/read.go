package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/mikebway/slog/s3"
	"github.com/spf13/cobra"
)

var (
	startDateStr   string         // flag value defining the start time of the window to be processed
	startDateTime  time.Time      // the start time of the window to be processed
	windowStr      string         // flag value defining the duration / time span to be considered
	window         time.Duration  // the duration / time span to be considered
	contentTypeStr string         // Specifies which fields are to be included in the log output
	contentType    s3.ContentType // Content type as an enumerated value

	// We build the parameters to be passed to he command execution
	// as a global so that they can be checked by unit test code
	slogSession *s3.SlogSession
)

// readCmd represents the read command
var readCmd = &cobra.Command{
	Use:   "read log-bucket [source-bucket*]",
	Short: "Display S3 hosted web logs for a given time window",
	Long: `Given a start date and time, together with a time window, displays the S3 hosted
web logs from a specified bucket for that time window. Optionally, filters the
log data to only include those entries that match the list of source buckets.`,

	RunE: func(cmd *cobra.Command, args []string) error {

		// There must be an S3 bucket name
		if len(args) == 0 {
			return errors.New("An S3 bucket name must be provided")
		}

		// Confirm that the content type requested is valid
		err := validateContentType()
		if err != nil {
			return err
		}

		// Parse the start time
		startDateTime, err = time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			return fmt.Errorf("Invalid start date time: %w", err)
		}

		// Parse the time window
		window, err = parseTimeWindow(windowStr)
		if err != nil {
			return fmt.Errorf("Invalid time window: %w", err)
		}

		// Populate the SlogSession to wrap our parameters up for the run
		slogSession = &s3.SlogSession{
			Region:        region,
			LogBucket:     args[0],
			Folder:        path,
			SourceBuckets: args[1:],
			StartDateTime: startDateTime,
			EndDateTime:   startDateTime.Add(window),
			Content:       contentType,
		}

		// All is well with the command formating and AWS access (to the best of our present knowledge).
		// Go ahead and do the work unless we are unit testing.
		fmt.Printf("Reading logs from %v/%v for with start=%v, window=%v seconds\n",
			args[0], path, startDateTime.Format(time.RFC3339), window.Seconds())
		if !unitTesting {
			err = s3.DisplayLog(slogSession)
		}
		if err != nil {
			// Placing the error check here rather than inside the !unitTesting block
			// increases unit test coverage without sacrificing integrity
			return err
		}

		// Command line parsing succeeded even if the execution failed
		return nil
	},
}

func init() {
	rootCmd.AddCommand(readCmd)

	// Initialize the flags that apply to the read command and, potentially, to subcommands
	initReadFlags()
}

// initRootFlags is called from init() to define the flags that apply to the read
// command, and might be inherited by its subcommands. It is defined separately from
// init() so that it can be invoked by unit tests when they need to reset the playing field.
func initReadFlags() {

	// Local flag definitions
	readCmd.Flags().StringVar(&startDateStr, "start", "2020-01-01T00:00:00-00:00",
		`Start date time in the form 2020-01-02T15:04:05Z07:00 form with time zone offset
`)
	readCmd.Flags().StringVar(&windowStr, "window", "1h",
		`Time window in the days (d), hours (h), minutes (m) or seconds (s).
For example '90s' for 90 seconds. '36h' for 36 hours.`)
	readCmd.Flags().StringVar(&contentTypeStr, "content", "basic",
		`Content to include in the log output; must be one of the following:
   basic     - minimal useful content, no bucket names, owners, request IDs etc
   requestid - includes the request ID
   bucket    - prefixed with the Web source bucket name (useful if capturing
               logs from multiple buckets into one location)
   rich      - includes bucket, request ID, operation and key values
   raw       - the whole enchilada, as originally recorded by AWS;
               ignores source bucket filtering; outputs all lines 
`)
}

// Parse a time window string into a duration
func parseTimeWindow(wstr string) (time.Duration, error) {

	// The string must be at least two characters in length
	l := len(wstr)
	if l > 1 {

		// The last character tells us the type of the number that precedes it (hours, minites, etc)
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

// validateContentType ensures that the content type provided, or its default, are
// valid log content types that we know how to render.
func validateContentType() error {

	switch contentTypeStr {
	case "basic":
		contentType = s3.BASIC
	case "requestid":
		contentType = s3.REQUESTID
	case "bucket":
		contentType = s3.BUCKET
	case "rich":
		contentType = s3.RICH
	case "raw":
		contentType = s3.RAW
	default:
		return fmt.Errorf("Unrecognized content type: %s", contentTypeStr)
	}

	// If we get to this point, all is well with our corner of the world
	return nil
}
