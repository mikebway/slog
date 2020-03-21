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

// A structure packing the various handles need to access S3 buckets
// and the parameters for a given run.
type awsAccess struct {
	session       *session.Session // The S3 session
	s3            *s3.S3           // The S3 client
	bucket        string           // The name of the target bucket
	folder        string           // The name of the folder to be walked within the bucket
	startDateTime time.Time        // When reading logs, the timestamp of the earliest entry sought
	endDateTime   time.Time        // When reading logs, the timestamp of the latest entry sought
}

// establishAWSAccess attempts to create an AWS session using the default
// access key and secret defined by the shell environment and/or confguration
// file. It then opens an S3 client using this session and returns the pair
// in a single structure that can be passed around to worker functions that
// need it.
func establishAWSAccess(region string) (*awsAccess, error) {

	// Request a session with the default credentials for the default region
	sess, err := session.NewSession(
		&aws.Config{
			Region: aws.String(region),
		},
	)
	if err != nil {
		fmt.Println("Error creating session: ", err)
		return nil, err
	}

	// Obtain an S3 service handle
	s3Client := s3.New(sess)

	// All good - return both wrapped with a bow
	return &awsAccess{
		session: sess,
		s3:      s3Client,
	}, nil
}

// fetchLogObjectKeys loops requesting pages of object keys starting from, approximately,
// the time given until there are no more keys or the keys fall outside the given
// time window (more recent than endDateTime). It posts those keys to keyChan. When there
// are no more keys fitting the time window to post, it closes keyChan and returns.
//
// If a problem occurs, fetchLogObjectKeys posts an error to errChan and terminates // returns
// after closing keyChan.
func fetchLogObjectKeys(access *awsAccess, keyChan chan<- string, errChan chan<- error) {

	// Form the folder prefix from the path provided
	prefix := access.folder + "/"

	// Format the start time to ther nearest minute and combine with the prefix
	// to form the "start after" key
	startAfter := prefix + access.startDateTime.UTC().Format("2006-01-02-15-04-05")

	// Calculate the key prefix that will signal we have reached the end
	endAfter := prefix + access.endDateTime.UTC().Format("2006-01-02-15-04-05")

	// Set up our starting point for paging through S3 bucket keynames
	input := &s3.ListObjectsV2Input{
		MaxKeys:    aws.Int64(maxListKeys),
		Bucket:     &access.bucket,
		Prefix:     &prefix,
		StartAfter: &startAfter,
	}

	// Ask for the object list, with a callback function to receive pages of data
	err := access.s3.ListObjectsV2Pages(input,
		func(page *s3.ListObjectsV2Output, lastPage bool) bool {

			// Loop through all the objects, sending their keys on to the next stage through keyChan
			for _, obj := range page.Contents {

				// Confirm that we have a valid key that is not the parent folder
				key := obj.Key
				if key == nil || *key == access.folder {
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
