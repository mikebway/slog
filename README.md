# slog - A CLI Utility for Managing S3 Static Web Site Logs

slog allows you to sequentially view access log data from web sites that are hosted
and served from Amazon S3 buckets. 

If you follow the [AWS directions for hosting static Web content](https://docs.aws.amazon.com/AmazonS3/latest/dev/WebsiteHosting.html),
and [configure traffic logging](https://docs.aws.amazon.com/AmazonS3/latest/dev/LoggingWebsiteTraffic.html),
the resulting log data is fragmented over a large number of S3 files/objects named for the time at which the access
they record was generated - typically one file per second of activity but potentially more in higher traffic
situations. Unless you pool these many log files into Splunk, or similar, they can be almost impossible to
make sense of. That's where slog comes in.

## Usage

Running the `slog` utility with either the `help` command or with no parameters at all will display the
following usage information:

```text
Slog is a CLI libraray for reading and culling web access logs stored in S3.

Typically, the logs managed are those generated in response to access to static web assets
themselves served directly from S3.

Usage:
  slog [command]

Available Commands:
  help        Help about any command
  read        Display S3 hosted web logs for a given time window

Flags:
  -h, --help   help for slog

Use "slog [command] --help" for more information about a command.
```

## Unit Testing

To run all of the unit tests from the command line and receive a coverage report:

```bash
go test -cover ./...
```
