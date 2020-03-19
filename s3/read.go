package s3

// The functions in this file deal with establishing an AWS session

import (
	"fmt"
	"time"
)

// DisplayLog prints the Web logs from the given bucket and root path / folder, from
// the given start time and for the given time window following that.
//
// An error is returned if there is a proble, otherwise nil.
func DisplayLog(bucket, folder string, startDateTime time.Time, window time.Duration) error {

	// Obtain an AWS session
	// session, err := establishAWSSession()
	// if err != nil {
	// 	return err
	// }

	// // Obtain an S3 service handle
	// s3Service := s3.New(session)

	// Establish the variaous communicatiomn channels that we will need
	errChan := make(chan error)      // Used to signal errors that require the app DisplayLog to terminate
	keyChan := make(chan string, 5)  // Distributes S3 object keys listed from the log bucket
	dataChan := make(chan []byte, 5) // DIstributes byte buffers downloaded from S3 objects
	doneChan := make(chan struct{})  // Used by the final display function to signal when it is finished

	// Ate what time does our window of interest close
	endDateTime := startDateTime.Add(window)

	// Spin up the function that lists keys from the bucket
	go fetchLogObjectKeys(bucket, folder, startDateTime, endDateTime, keyChan, errChan)

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
func fetchLogObjectKeys(bucket, folder string, startDateTime time.Time, endDateTime time.Time, keyChan chan<- string, errChan chan<- error) {

	// Just testing the design - not real code
	keyChan <- "Bucket: " + bucket
	keyChan <- "Folder: " + folder
	keyChan <- "Start: " + startDateTime.Format(time.RFC3339)
	keyChan <- "End: " + endDateTime.Format(time.RFC3339)
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
