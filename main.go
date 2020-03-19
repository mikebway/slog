// A command line utility for managing web access logs stored in S3,
// typically those logs collected from static Web artifacts served from S3.
package main

import "github.com/mikebway/slog/cmd"

// Command line entry point.
//
// Cobra based command line parsing does all the work
func main() {
	cmd.Execute()
}
