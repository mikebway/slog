package s3

// The functions in this file deal with establishing an AWS session

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

// establishAWSSession attempts to create an AWS session using the default
// access key and secret defined by the shell environment and/or confguration
// file.
func establishAWSSession(region string) (*session.Session, error) {

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
	return sess, nil
}
