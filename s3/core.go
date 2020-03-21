package s3

// The functions in this file deal with establishing an AWS session, S3 access,
// and some functions common to read and delete operations.

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// SlogSession is a structure packing the various parameters for a given run.
type SlogSession struct {
	awsSession    *session.Session // The S3 session
	s3            *s3.S3           // The S3 client
	Region        string           // The AWS region where the S3 bucket is hosted
	Bucket        string           // The name of the target bucket
	Folder        string           // The name of the folder to be walked within the bucket
	StartDateTime time.Time        // When reading logs, the timestamp of the earliest entry sought
	EndDateTime   time.Time        // When reading logs, the timestamp of the latest entry sought
}

// activateSession adds an AWS session and and S3 client to a SlogSession
// if they are not already populated.
//
// If all goes well, returns nil, otherwise an error.
func activateSession(slogSession *SlogSession) error {

	// If the session has already been actived, we have nothing to do
	if slogSession.s3 != nil {
		return nil
	}

	// Request a session with the default credentials for the default region
	awsSession, err := session.NewSession(
		&aws.Config{
			Region: &slogSession.Region,
		},
	)
	if err != nil {
		fmt.Println("Error creating session: ", err)
		return err
	}

	// Obtain an S3 service handle
	s3Client := s3.New(awsSession)

	// All good - put those in the session and return happy
	slogSession.awsSession = awsSession
	slogSession.s3 = s3Client
	return nil
}

// fetchLogObjectKeys loops requesting pages of object keys starting from, approximately,
// the time given until there are no more keys or the keys fall outside the given
// time window (more recent than endDateTime). It posts those keys to keyChan. When there
// are no more keys fitting the time window to post, it closes keyChan and returns.
//
// If a problem occurs, fetchLogObjectKeys posts an error to errChan and terminates // returns
// after closing keyChan.
func fetchLogObjectKeys(session *SlogSession, keyChan chan<- string, errChan chan<- error) {

	// Form the folder prefix from the path provided
	prefix := session.Folder + "/"

	// Format the start time to ther nearest minute and combine with the prefix
	// to form the "start after" key
	startAfter := prefix + session.StartDateTime.UTC().Format("2006-01-02-15-04-05")

	// Calculate the key prefix that will signal we have reached the end
	endAfter := prefix + session.EndDateTime.UTC().Format("2006-01-02-15-04-05")

	// Set up our starting point for paging through S3 bucket keynames
	input := &s3.ListObjectsV2Input{
		MaxKeys:    aws.Int64(maxListKeys),
		Bucket:     &session.Bucket,
		Prefix:     &prefix,
		StartAfter: &startAfter,
	}

	// Ask for the object list, with a callback function to receive pages of data
	err := session.s3.ListObjectsV2Pages(input,
		func(page *s3.ListObjectsV2Output, lastPage bool) bool {

			// Loop through all the objects, sending their keys on to the next stage through keyChan
			for _, obj := range page.Contents {

				// Confirm that we have a valid key that is not the parent folder
				key := obj.Key
				if key == nil || *key == session.Folder {
					continue
				}

				// Test if the key is beyond our end time
				if *key > endAfter {

					// we are done - stop paging now
					return false
				}

				// Pass the key down the processing chain
				keyChan <- *key
			}

			// Go round for the next page if there is one still to come
			return !lastPage
		})
	if err != nil {
		// The ListObjectsV2Pages request failed, report the error
		errChan <- err
	}

	// We are done - close the key channel
	close(keyChan)
}
