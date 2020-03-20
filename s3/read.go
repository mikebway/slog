package s3

// The functions in this file deal with establishing an AWS session

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// DisplayLog prints the Web logs from the given bucket and root path / folder, from
// the given start time and for the given time window following that.
//
// An error is returned if there is a proble, otherwise nil.
func DisplayLog(region, bucket, folder string, startDateTime time.Time, window time.Duration) error {

	// Obtain an AWS session
	session, err := establishAWSSession(region)
	if err != nil {
		return err
	}

	// Obtain an S3 service handle
	s3Client := s3.New(session)

	// Establish the various communicatiomn channels that we will need
	errChan := make(chan error)      // Used to signal errors that require the app DisplayLog to terminate
	keyChan := make(chan string, 5)  // Distributes S3 object keys listed from the log bucket
	dataChan := make(chan []byte, 5) // DIstributes byte buffers downloaded from S3 objects
	doneChan := make(chan struct{})  // Used by the final display function to signal when it is finished

	// At what time does our window of interest close
	endDateTime := startDateTime.Add(window)

	// Spin up the function that lists keys from the bucket
	go fetchLogObjectKeys(s3Client, bucket, folder, startDateTime, endDateTime, keyChan, errChan)

	// Spin up the data fetching function that consumes those keys and pulls down the object content
	go fetchLogObjectData(bucket, keyChan, dataChan, errChan)

	// Spin up the data display function
	go displayLogData(endDateTime, dataChan, doneChan, errChan)

	// Wait until we are done or see an error
	select {
	case <-doneChan:
		return nil
	case err := <-errChan:
		return err
	}
}

// fetchLogObjectKeys loops requesting pages of object keys starting from, approximately,
// the time given until there are no more keys or the keys fall outside the given
// time window (more recent than endDateTime). It posts those keys to keyChan. When there
// are no more keys fitting the time window to post, it closes keyChan and returns.
//
// If a problem occurs, fetchLogObjectKeys posts an error to errChan and terminates // returns
// after closing keyChan.
func fetchLogObjectKeys(s3Client *s3.S3, bucket, folder string, startDateTime time.Time, endDateTime time.Time, keyChan chan<- string, errChan chan<- error) {

	// Form the folder prefix from the path provided
	prefix := folder + "/"

	// Format the start time to ther nearest minute and combine with the prefix
	// to form the "start after" key
	startAfter := prefix + startDateTime.UTC().Format("2006-01-02-15-04-05")

	// Set up our starting point for paging through S3 bucket keynames
	input := &s3.ListObjectsV2Input{
		MaxKeys:    aws.Int64(10),
		Bucket:     &bucket,
		Prefix:     &prefix,
		StartAfter: &startAfter,
	}

	// Just testing the design - not real code
	pageNum := 0

	// Ask for the object list, with a callback function to receive pages of data
	err := s3Client.ListObjectsV2Pages(input,
		func(page *s3.ListObjectsV2Output, lastPage bool) bool {
			pageNum++

			// Loop through all the objects, sending their keys on to the next stage through keyChan
			for _, obj := range page.Contents {
				if obj.Key == nil || *obj.Key == folder {
					continue
				}
				keyChan <- *obj.Key
			}
			return pageNum <= 3
		})
	if err != nil {
		// The ListObjectsV2Pages request failed, report the error
		errChan <- err
	}

	// We are done - close the key channel
	close(keyChan)
}

// fetchLogObjectData listens to keyChan for kyes, downloaads the content of the corresponding
// S3 objects to in memory buffers, then writes those buffers to dataChan. When keyChan is closed,
// fetchLogObjectData closes dataChan and returns.
//
// If a problem occurs, fetchLogObjectData posts an error to errChan and terminates // returns after closing
// dataChan.
func fetchLogObjectData(bucket string, keyChan <-chan string, dataChan chan<- []byte, errChan chan<- error) {

	// Just testing the design - not real code
	for key := range keyChan {
		dataChan <- []byte(key)
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
func displayLogData(endDateTime time.Time, dataChan <-chan []byte, doneChan chan<- struct{}, errChan chan<- error) {

	// Just testing the design - not real code
	for buff := range dataChan {
		fmt.Println(string(buff))
	}
	close(doneChan)
}
