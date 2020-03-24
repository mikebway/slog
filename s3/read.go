package s3

// The functions in this file deal with establishing an AWS session

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var (
	maxListKeys int64 = 100 // Max number of keys to fetch per page; can be overridden for unit testing
)

// DisplayLog prints the Web logs from the bucket and root path / folder, between
// the start and end times, defined in the given session structure.
//
// An error is returned if there is a proble, otherwise nil.
func DisplayLog(session *SlogSession) error {

	// Populate the session with AWS session and client handles
	err := activateSession(session)
	if err != nil {
		return err
	}

	// Establish the various communicatiomn channels that we will need
	errChan := make(chan error)                  // Used to signal errors that require the app DisplayLog to terminate
	keyChan := make(chan string, 5)              // Distributes S3 object keys listed from the log bucket
	dataChan := make(chan *aws.WriteAtBuffer, 5) // Distributes AWS wrapped byte buffers downloaded from S3 objects
	doneChan := make(chan struct{})              // Used by the final display function to signal when it is finished

	// Spin up the function that lists keys from the bucket
	go fetchLogObjectKeys(session, keyChan, errChan)

	// Spin up the data fetching function that consumes those keys and pulls down the object content
	go fetchLogObjectData(session, keyChan, dataChan, errChan)

	// Spin up the data display function
	go displayLogData(session, dataChan, doneChan, errChan)

	// Wait until we are done or see an error
	select {
	case <-doneChan:
		return nil
	case err := <-errChan:
		return err
	}
}

// fetchLogObjectData listens to keyChan for kyes, downloaads the content of the corresponding
// S3 objects to in memory buffers, then writes those buffers to dataChan. When keyChan is closed,
// fetchLogObjectData closes dataChan and returns.
//
// If a problem occurs, fetchLogObjectData posts an error to errChan and terminates // returns after closing
// dataChan.
func fetchLogObjectData(session *SlogSession, keyChan <-chan string, dataChan chan<- *aws.WriteAtBuffer, errChan chan<- error) {

	// Establish a download manager
	downloader := s3manager.NewDownloaderWithClient(session.s3)

	// For all the keys we get through the channel ...
	for key := range keyChan {

		// We download to a buffer, not a file, using a buffer writer
		awsBuff := &aws.WriteAtBuffer{}

		// Download the object
		_, err := downloader.Download(awsBuff,
			&s3.GetObjectInput{
				Bucket: aws.String(session.Bucket),
				Key:    aws.String(key),
			})

		// If that did not work -- post an error back to our caller
		// and exit the key reading loop to close the data channel
		if err != nil {
			errChan <- err
			break
		}

		// Send the buffer we just got on down the pipeline
		dataChan <- awsBuff
	}
	close(dataChan)
}

// displayLogData listens to dataChan, rendering the buffers that it receives to the display as lines
// unitl the channel is closed.
//
// Once the end of the data is encountered and displayed, displayLogData closes doneChan to signal
// that the job is complete.
//
// If a problem occurs, displayLogData posts an error to errChan and returns without closing doneChan.
func displayLogData(session *SlogSession, dataChan <-chan *aws.WriteAtBuffer, doneChan chan<- struct{}, errChan chan<- error) {

	// Process each buffer delivered through dataChan
	for awsBuff := range dataChan {

		// Displaying raw data requires much less processing than selective log output
		// so we handle that separately and here, in a tighter loop
		if session.Content == RAW {

			// AWS Web log objects end with a newline character so no need to "Println()"
			fmt.Print(string(awsBuff.Bytes()))
			continue
		}

		// Not displaying raw log content ...
		// We have to break up the buffer and manipulate the lines that it contains
		err := displaySelectLogData(session, awsBuff)
		if err != nil {
			errChan <- err
			return
		}
	}
	close(doneChan)
}

// displaySelectLogData eliminates cruft from the raw AWS web log data and displays a subset of the
// fields contained in each line, as dictated by the SlogSession.Content value.
func displaySelectLogData(session *SlogSession, awsBuff *aws.WriteAtBuffer) error {

	// Break the buffer into lines that we can evaluate
	lines := strings.Split(string(awsBuff.Bytes()), "\n")

	// Loop over the lines, applying the requested treatment
	for _, line := range lines {

		// Skip blank lines
		if len(line) == 0 {
			continue
		}

		// Process the line based on the content type requested
		switch session.Content {
		case BASIC:
			line = basicContent(line)
		case REQUESTID:
			line = requestContent(line)
		case BUCKET:
			line = bucketContent(line)
		case RICH:
			line = richContent(line)
		default:
			return fmt.Errorf("No implementation for content type: %d", session.Content)
		}

		// Display the treated (or untreated) line
		fmt.Println(line)
	}

	return nil
}

// basicContent returns the least amount of information from raw AWS web log entries, typically
// more than enough to be useful without filling the screen with noise.
func basicContent(line string) string {

	// Split the line into words / fields. This is problematic since some fields actually contain spaces :-(
	parts := strings.Split(line, " ")

	// Build up parts from consecutive runs of fields that we want. The problem lies
	// with the User-Agent field that will contain a variable number of spaces and thus generate
	// a variable number of parts. We over come this by slicing the parts from the start of the User-Agent
	// to a count back from the end of parts we do not want at the end of the line.
	count := len(parts)
	part1 := strings.Join(parts[2:5], " ")
	part2 := strings.Join(parts[9:count-7], " ")

	// Add the parts together and return
	return part1 + " " + part2
}

// requestContent returns the basic content plus the Amazon generated request ID.
func requestContent(line string) string {

	// See algorithm comments in basicContent(..)
	parts := strings.Split(line, " ")
	count := len(parts)
	part1 := strings.Join(parts[2:5], " ")
	requestID := parts[6]
	part2 := strings.Join(parts[9:count-7], " ")

	// Add the parts together and return
	return part1 + " " + requestID + " " + part2
}

// bucketContent returns the the basic content plus the name of the S3 bucket that it was served from.
// This is useful if the log bucket is being used to collect Web log data associated with multiple
// buckets, for example where blog pages are served out of one bucket but images or Javascript
// files are served from another.
func bucketContent(line string) string {

	// See algorithm comments in basicContent(..)
	parts := strings.Split(line, " ")
	count := len(parts)
	part1 := strings.Join(parts[1:5], " ")
	part2 := strings.Join(parts[9:count-7], " ")

	// Add the parts together and return
	return part1 + " " + part2
}

// richContent returns most of the data from the log entry but excludes distracting noise like
// the AWS ID for bucket owner etc. These take up a lot of space and are not typically of interest
// to Web site managers.
func richContent(line string) string {

	// See algorithm comments in basicContent(..)
	parts := strings.Split(line, " ")
	count := len(parts)
	part1 := strings.Join(parts[1:5], " ")
	part2 := strings.Join(parts[6:count-7], " ")

	// Add the parts together and return
	return part1 + " " + part2
}
