package logs

import (
	"encoding/json"
	_ "encoding/xml"
	"fmt"
	"github.com/crowdmob/goamz/aws"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

const debug = false

type CloudWatchLogs struct {
	Service  aws.AWSService
	Endpoint string
	Auth     aws.Auth
	Region   aws.Region
}

// Create a new CloudWatchLogs object for a given namespace
func New(auth aws.Auth, region aws.ServiceInfo) (*CloudWatchLogs, error) {
	service, err := aws.NewService(auth, region)
	if err != nil {
		return nil, err
	}
	return &CloudWatchLogs{
		Service:  service,
		Endpoint: region.Endpoint,
		Auth:     auth,
		Region:   aws.Region{Name: "us-east-1"},
	}, nil
}

func buildError(r *http.Response, jsonBody []byte) error {
	kinesisError := &Error{
		StatusCode: r.StatusCode,
		Status:     r.Status,
	}

	err := json.Unmarshal(jsonBody, kinesisError)
	if err != nil {
		log.Printf("Failed to parse body as JSON")
		return err
	}

	return kinesisError
}

// Error represents an error in an operation with Kinesis(following goamz/Dynamodb)
type Error struct {
	StatusCode int // HTTP status code (200, 403, ...)
	Status     string
	Code       string `json:"__type"`
	Message    string `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("[HTTP %d] %s : %s\n", e.StatusCode, e.Code, e.Message)
}

func (k *CloudWatchLogs) query(target string, payload interface{}) ([]byte, error) {

	jsonBody, _ := json.Marshal(payload)

	data := strings.NewReader(string(jsonBody))
	hreq, err := http.NewRequest("POST", k.Endpoint+"/", data)

	if err != nil {
		return nil, err
	}

	hreq.Header.Set("Content-Type", "application/x-amz-json-1.1")
	hreq.Header.Set("X-Amz-Date", time.Now().UTC().Format(aws.ISO8601BasicFormat))
	hreq.Header.Set("X-Amz-Target", target)

	signer := aws.NewV4Signer(k.Auth, "logs", k.Region)
	signer.Sign(hreq)

	resp, err := http.DefaultClient.Do(hreq)

	if err != nil {
		log.Printf("Error calling Amazon\n: %v", err)
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Could not read response body\n")
		return nil, err
	}

	if resp.StatusCode != 200 {
		err = buildError(resp, body)
		return nil, err
	}

	return body, nil
}

type LogGroup struct {
	Arn               string
	CreationTime      int
	LogGroupName      string
	MetricFilterCount int
	RetentionInDays   int
	StoredBytes       int
}

type DescribeLogGroupsResponse struct {
	NextToken string
	LogGroups []*LogGroup
}

type DescribeLogGroupsRequest struct {
	Limit              string
	LogGroupNamePrefix string
	NextToken          string
}

func (c *CloudWatchLogs) DescribeLogGroups(req *DescribeLogGroupsRequest) (r []*LogGroup, err error) {
	data, err := c.query("Logs_20140328.DescribeLogGroups", req)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}
	var result DescribeLogGroupsResponse
	json.Unmarshal(data, &result)
	return result.LogGroups, nil
}

type LogStream struct {
	Arn                 string
	CreationTime        int
	FirstEventTimestamp int
	LastEventTimestamp  int
	LastIngestionTime   int
	LogStreamName       string
	StoredBytes         int
	UploadSequenceToken string
}

type DescribeLogStreamsResponse struct {
	NextToken  string
	LogStreams []*LogStream
}

type DescribeLogStreamsRequest struct {
	Limit               int    `json:"limit,omitempty"`
	LogGroupName        string `json:"logGroupName,omitempty"`
	LogStreamNamePrefix string `json:"logStreamNamePrefix,omitempty"`
	NextToken           string `json:"nextToken,omitempty"`
}

func (c *CloudWatchLogs) DescribeLogStreams(req *DescribeLogStreamsRequest) (r []*LogStream, err error) {
	data, err := c.query("Logs_20140328.DescribeLogStreams", req)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}
	var result DescribeLogStreamsResponse
	json.Unmarshal(data, &result)
	if len(result.NextToken) > 0 {
		req.NextToken = result.NextToken
		return c.DescribeLogStreams(req)
	}
	return result.LogStreams, nil
}

type Event struct {
	IngestionTime int
	Message       string
	Timestamp     int
}

type GetLogEventsResponse struct {
	Events            []*Event
	NextForwardToken  string
	NextBackwardToken string
}

type GetLogEventsRequest struct {
	Limit         int    `json:"limit,omitempty"`
	EndTime       int    `json:"endTime,omitempty"`
	LogGroupName  string `json:"logGroupName,omitempty"`
	LogStreamName string `json:"logStreamName,omitempty"`
	NextToken     string `json:"nextToken,omitempty"`
	StartFromHead bool   `json:"startFromHead"`
	StartTime     int    `json:"startTime"`
}

func (c *CloudWatchLogs) GetLogEvents(req *GetLogEventsRequest) (r *GetLogEventsResponse, err error) {
	data, err := c.query("Logs_20140328.GetLogEvents", req)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}
	var result GetLogEventsResponse
	json.Unmarshal(data, &result)
	return &result, nil
}
