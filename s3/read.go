package s3

// The functions in this file deal with establishing an AWS session

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var (
	maxListKeys int64 = 100 // Max number of keys to fetch per page; override for unit testing
)

// DisplayLog prints the Web logs from the given bucket and root path / folder, from
// the given start time and for the given time window following that.
//
// An error is returned if there is a proble, otherwise nil.
func DisplayLog(region, bucket, folder string, startDateTime time.Time, window time.Duration) error {

	// Obtain an access package populated with AWS session and S3 client
	access, err := establishAWSAccess(region)
	if err != nil {
		return err
	}

	// Fill in the rest of the access structure
	access.bucket = bucket
	access.folder = folder
	access.startDateTime = startDateTime
	access.endDateTime = startDateTime.Add(window)

	// Establish the various communicatiomn channels that we will need
	errChan := make(chan error)                  // Used to signal errors that require the app DisplayLog to terminate
	keyChan := make(chan string, 5)              // Distributes S3 object keys listed from the log bucket
	dataChan := make(chan *aws.WriteAtBuffer, 5) // Distributes AWS wrapped byte buffers downloaded from S3 objects
	doneChan := make(chan struct{})              // Used by the final display function to signal when it is finished

	// Spin up the function that lists keys from the bucket
	go fetchLogObjectKeys(access, keyChan, errChan)

	// Spin up the data fetching function that consumes those keys and pulls down the object content
	go fetchLogObjectData(access, keyChan, dataChan, errChan)

	// Spin up the data display function
	go displayLogData(access, dataChan, doneChan, errChan)

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
func fetchLogObjectData(access *awsAccess, keyChan <-chan string, dataChan chan<- *aws.WriteAtBuffer, errChan chan<- error) {

	// Establish a download manager
	downloader := s3manager.NewDownloaderWithClient(access.s3)

	// For all the keys we get through the channel ...
	for key := range keyChan {

		// We download to a buffer, not a file, using a buffer writer
		awsBuff := &aws.WriteAtBuffer{}

		// Download the object
		_, err := downloader.Download(awsBuff,
			&s3.GetObjectInput{
				Bucket: aws.String(access.bucket),
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

// displayLogData listens to dataChan, rendering the buffers that it receives to the display as strings
// unitl either the channel is closed or it encounters a log entry that is newer than endDateTime.
//
// Once the end of the data is encountered and displayed, displayLogData closes doneChan to signal
// that the job is complete.
//
// If a problem occurs, fetchLogObjectData posts an error to errChan and returns without closing doneChan.
func displayLogData(access *awsAccess, dataChan <-chan *aws.WriteAtBuffer, doneChan chan<- struct{}, errChan chan<- error) {

	// Just testing the design - not real code
	for awsBuff := range dataChan {
		fmt.Println(string(awsBuff.Bytes()))
	}
	close(doneChan)
}
