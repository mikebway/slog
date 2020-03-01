// A command line utility for managing web access logs stored in S3,
// typically those logs collected from static Web artifacts served from S3.
package main

import "github.com/mikebway/slog/cmd"

func main() {
	cmd.Execute()
}
