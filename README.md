# goamz / logs

builds on top of [github.com/crowdmob/goamz](https://github.com/crowdmob/goamz) adding support for [Amazon CloudWatch Logs](http://docs.aws.amazon.com/AmazonCloudWatch/latest/DeveloperGuide/WhatIsCloudWatchLogs.html)

## What's implemented

Basically only what's needed in order to monitor Amazon CloudWatch logs

* fetch group descriptors
* fetch stream descriptors
* fetch stream events

## How to build and install

Just use `go get`. For example:

* `$ go get github.com/creamdog/goamz/logs`


## Usage Example, Reading

```go
package main

import (
	"fmt"
	"github.com/creamdog/goamz/logs"
	"github.com/crowdmob/goamz/aws"
	"time"
)

//
// EXAMPLE: fetches and prints latest log entries from all streams in all log groups
//
func main() {
	auth := aws.Auth{AccessKey: "<ACCESS KEY>", SecretKey: "<SECRET KEY>"}
	client, err := logs.New(auth, "https://logs.us-east-1.amazonaws.com", "us-east-1")
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	// fetch group descriptors
	groups, err := client.DescribeLogGroups(&logs.DescribeLogGroupsRequest{})
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	for _, group := range groups {

		// fetch group stream descriptors
		streams, err := client.DescribeLogStreams(&logs.DescribeLogStreamsRequest{
			LogGroupName: group.LogGroupName,
		})
		if err != nil {
			fmt.Printf("%v\n", err)
			continue
		}
		for _, stream := range streams {

			//fetch latest stream events
			events, err := client.GetLogEvents(&logs.GetLogEventsRequest{
				LogGroupName:  group.LogGroupName,
				LogStreamName: stream.LogStreamName,
				//StartFromHead: false,
				//StartTime: 1234564561651, // UNIX TIMESTAMP
				//EndTime: 651651561651, // UNIX TIMESTAMP
				//Limit: 100,
			})
			if err != nil {
				fmt.Printf("%v\n", err)
				continue
			}

			for _, event := range events.Events {
				fmt.Printf("%s / %s [%s] %s\n", group.LogGroupName, stream.LogStreamName, time.Unix(event.Timestamp/1000, 0).Format(time.RFC3339), event.Message)
			}
		}
	}

}
```

## Usage Example, Writing

```go
package main

import (
	"fmt"
	"github.com/creamdog/goamz/logs"
	"github.com/crowdmob/goamz/aws"
	"os"
	"time"
)

//
// creates a log group named "golang.test", if it doesn't allready exist.
// creates a log stream named using the machine hostname, if it doesn't allready exist.
// send 10 batches of logs with 100 events each into the log stream
//
func main() {

	logGroupName := "golang.test"
	logStreamName, err := os.Hostname()
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}

	// initialize cloudwatch log client, group and stream
	client, sequenceToken, err := initialize(logGroupName, logStreamName)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}

	// log some stuff
	for i := 0; i < 10; i++ {

		// create some log events
		events := make([]logs.LogEvent, 0)
		for e := 0; e < 100; e++ {
			events = append(events, logs.LogEvent{Message: time.Now().Format("2006-01-02 15:04:05") + " hello", Timestamp: time.Now().UnixNano() / 1000000})
		}

		// send log events
		sequenceToken, err = client.PutLogEvents(&logs.PutLogEventsRequest{
			LogEvents:     events,
			LogStreamName: logStreamName,
			LogGroupName:  logGroupName,
			SequenceToken: sequenceToken})

		if err != nil {
			fmt.Printf("%s\n", err)
			return
		}
	}

}

func initialize(logGroupName, logStreamName string) (*logs.CloudWatchLogs, string, error) {
	auth := aws.Auth{AccessKey: "<ACCESS KEY>", SecretKey: "<SECRET KEY>"}
	client, err := logs.New(auth, "https://logs.us-east-1.amazonaws.com", "us-east-1")
	if err != nil {
		return nil, "", err
	}

	nextToken := ""

	// fetch group descriptors
	groups, err := client.DescribeLogGroups(&logs.DescribeLogGroupsRequest{})
	if err != nil {
		return nil, "", err
	}

	// check if group exists
	groupExists := false
	for _, group := range groups {
		if group.LogGroupName == logGroupName {
			groupExists = true
			break
		}
	}
	// create group if it does not exist
	if !groupExists {
		if err := client.CreateLogGroup(&logs.CreateLogGroupRequest{logGroupName}); err != nil {
			return nil, "", err
		}
	}

	// fetch group stream descriptors
	streams, err := client.DescribeLogStreams(&logs.DescribeLogStreamsRequest{
		LogGroupName: logGroupName,
	})
	if err != nil {
		return nil, "", err
	}

	// check if stream exists
	streamExists := false
	for _, stream := range streams {
		if stream.LogStreamName == logStreamName {
			nextToken = stream.UploadSequenceToken
			streamExists = true
			break
		}
	}
	// create stream if it doesn't exist
	if !streamExists {
		if err := client.CreateLogStream(&logs.CreateLogStreamRequest{logGroupName, logStreamName}); err != nil {
			return nil, "", err
		}
	}

	return client, nextToken, nil
}

```