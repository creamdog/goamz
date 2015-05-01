package logs

import (
	"encoding/json"
	_ "encoding/xml"
	"fmt"
	"github.com/crowdmob/goamz/aws"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

const debug = false

type CloudWatchLogs struct {
	Endpoint string
	Auth     aws.Auth
	Region   aws.Region
}

// Create a new CloudWatchLogs object for a given namespace
func New(auth aws.Auth, endpoint string, region string) (*CloudWatchLogs, error) {
	return &CloudWatchLogs{
		Endpoint: endpoint,
		Auth:     auth,
		Region:   aws.Region{Name: region},
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
	CreationTime      int64
	LogGroupName      string
	MetricFilterCount int
	RetentionInDays   int
	StoredBytes       int64
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
	CreationTime        int64
	FirstEventTimestamp int64
	LastEventTimestamp  int64
	LastIngestionTime   int64
	LogStreamName       string
	StoredBytes         int64
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
		tmp, err := c.DescribeLogStreams(req)
		if err != nil {
			return nil, err
		}
		result.LogStreams = append(result.LogStreams, tmp...)
	}
	return result.LogStreams, nil
}

type Event struct {
	IngestionTime int
	Message       string
	Timestamp     int64
}

type GetLogEventsResponse struct {
	Events            []*Event
	NextForwardToken  string
	NextBackwardToken string
}

type GetLogEventsRequest struct {
	Limit         int    `json:"limit,omitempty"`
	EndTime       int64  `json:"endTime,omitempty"`
	LogGroupName  string `json:"logGroupName,omitempty"`
	LogStreamName string `json:"logStreamName,omitempty"`
	NextToken     string `json:"nextToken,omitempty"`
	StartFromHead bool   `json:"startFromHead"`
	StartTime     int64  `json:"startTime"`
}

func (c *CloudWatchLogs) GetLogEvents(req *GetLogEventsRequest) (r *GetLogEventsResponse, err error) {
	data, err := c.query("Logs_20140328.GetLogEvents", req)
	if err != nil {
		return nil, err
	}
	var result GetLogEventsResponse
	json.Unmarshal(data, &result)
	return &result, nil
}

type CreateLogGroupRequest struct {
	LogGroupName string `json:"logGroupName"`
}

func (c *CloudWatchLogs) CreateLogGroup(req *CreateLogGroupRequest) error {
	_, err := c.query("Logs_20140328.CreateLogGroup", req)
	if err != nil {
		return err
	}
	return nil
}

type CreateLogStreamRequest struct {
	LogGroupName  string `json:"logGroupName"`
	LogStreamName string `json:"logStreamName"`
}

func (c *CloudWatchLogs) CreateLogStream(req *CreateLogStreamRequest) error {
	_, err := c.query("Logs_20140328.CreateLogStream", req)
	if err != nil {
		return err
	}
	return nil
}

type LogEvent struct {
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

type ByTimestamp []LogEvent

func (events ByTimestamp) Len() int           { return len(events) }
func (events ByTimestamp) Swap(i, j int)      { events[i], events[j] = events[j], events[i] }
func (events ByTimestamp) Less(i, j int) bool { return events[i].Timestamp < events[j].Timestamp }

type PutLogEventsRequest struct {
	LogEvents     []LogEvent `json:"logEvents"`
	LogGroupName  string     `json:"logGroupName"`
	LogStreamName string     `json:"logStreamName"`
	SequenceToken string     `json:"sequenceToken,omitempty"`
}

type PutLogEventsResponse struct {
	NextSequenceToken string `json:"nextSequenceToken"`
}

func (c *CloudWatchLogs) PutLogEvents(req *PutLogEventsRequest) (string, error) {
	sort.Sort(ByTimestamp(req.LogEvents))
	data, err := c.query("Logs_20140328.PutLogEvents", req)
	if err != nil {
		return "", err
	}
	var result PutLogEventsResponse
	json.Unmarshal(data, &result)
	return result.NextSequenceToken, nil
}
