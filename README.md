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


## How to use example

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
				return
			}

			for _, event := range events.Events {
				fmt.Printf("%s / %s [%s] %s\n", group.LogGroupName, stream.LogStreamName, time.Unix(event.Timestamp/1000, 0).Format(time.RFC3339), event.Message)
			}
		}
	}

}
```
