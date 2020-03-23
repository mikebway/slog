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
Slog is a CLI utility for reading and culling web access logs stored in S3.

Typically, the logs managed are those generated in response to access to static web assets
themselves served directly from S3.

Usage:
  slog [command]

Available Commands:
  help        Help about any command
  read        Display S3 hosted web logs for a given time window

Flags:
  -h, --help            help for slog
      --path string     The path of the log data within the S3 bucket (default "root")
      --region string   the aws region to target (default "us-east-1")

Use "slog [command] --help" for more information about a command.
```

## What's Missing

I intend to add a `delete` command at some point, to clear out old logs up to a
given date. To unit/integration test without risking real log data I would need to
write fake log data to a safe bucket location. My own web site sees so little
traffic that I lack the incentive to roll my selaves up and get all that done yet.

The `read` command displays raw log data exactly as written by Amazon. This is
noisy and contains much that has little value while filling the screen. While
preserving the ability to see this raw data if explicitly requested, I would
like the default display to be slimmer and more efficient.

## Unit / Integration Testing

The unit tests are really more like integration tests in that they will invoke
AWS S3 API calls. This requires access to an S3 bucket with the web logs from
S3 hosted web content. If the parameters for this log bucket were hard coded
in the unit tests, nobody but the oroiginal auther would be able to run the
tests so these parameters can be set through environment variables as follows:

```bash
export SLOG_TEST_REGION=us-east-1
export SLOG_TEST_BUCKET=log.mikebroadway.com
export SLOG_TEST_FOLDER=root
export SLOG_TEST_START_DATETIME=2020-03-20T13:30:00Z
export SLOG_TEST_END_DATETIME=2020-03-20T14:00:00Z
export SLOG_TEST_CONTAINS="AA960FCC76F5673E WEBSITE.GET.OBJECT robots.txt"
```

The `SLOG_TEST_CONTAINS` varibale must describe a portion of the log content
that will be found somewhere between the start and end date and time given in the
other variables.

With the environment variables set, you can run all of the unit tests from the
command line and receive a coverage report:

```bash
go test -cover ./...
```

To ensure that all tests are run, and that none are assumed unchanged for the
cache of a previous run, you may add the `-count=1` flag to required that all
tests are run at least and exactly once:

```bash
go test -cover -count=1 ./...
```
