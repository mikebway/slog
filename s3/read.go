package s3

// The functions in this file deal with establishing an AWS session

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var (
	maxListKeys int64 = 100 // Max number of keys to fetch per page; override for unit testing
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
	go displayLogData(dataChan, doneChan, errChan)

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
// If a problem occurs, fetchLogObjectData posts an error to errChan and returns without closing doneChan.
func displayLogData(dataChan <-chan *aws.WriteAtBuffer, doneChan chan<- struct{}, errChan chan<- error) {

	// Just testing the design - not real code
	for awsBuff := range dataChan {

		// AWS Web log objects end with a newline character so no need to "Println()"
		fmt.Print(string(awsBuff.Bytes()))
	}
	close(doneChan)
}
