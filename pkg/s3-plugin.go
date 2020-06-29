package main

import (
	"fmt"
	"time"
	"context"
	"encoding/json"
	"net/http"
	"bytes"
	"io"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/tobgu/qframe"
	"github.com/KamalGalrani/dateparse"
)

// newDatasource returns datasource.ServeOpts.
func newDatasource() datasource.ServeOpts {
	// creates a instance manager for your plugin. The function passed
	// into `NewInstanceManger` is called when the instance is created
	// for the first time or when a datasource configuration changed.
	im := datasource.NewInstanceManager(newDataSourceInstance)
	ds := &S3DataSource{
		im: im,
	}

	return datasource.ServeOpts{
		QueryDataHandler:   ds,
		CheckHealthHandler: ds,
	}
}

type S3DataSource struct {
	// The instance manager can help with lifecycle management
	// of datasource instances in plugins. It's not a requirements
	// but a best practice that we recommend that you follow.
	im instancemgmt.InstanceManager

	svc      *s3.S3
	settings struct {
		Bucket    string `json:"bucket" binding:"Required"`
		Region    string `json:"region" binding:"Required"`
		AccessKey string `json:"accessKey"`
	}
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifer).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (td *S3DataSource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {

	// Unmarshal the json into our queryModel
	if err := json.Unmarshal(req.PluginContext.DataSourceInstanceSettings.JSONData, &td.settings); err != nil {
		return nil, err
	}

	config := aws.Config{
		Region: aws.String(td.settings.Region),
	}

	if td.settings.AccessKey != "" {
		if secretKey, found := req.PluginContext.DataSourceInstanceSettings.DecryptedSecureJSONData["secretKey"]; found {
			config.Credentials = credentials.NewStaticCredentials(td.settings.AccessKey, secretKey, "")
		}
	}

	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	td.svc = s3.New(sess, &config)

	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		res := td.query(ctx, q)

		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

type queryModel struct {
	Path        string `json:"path"`
	Format      string `json:"format"`
	Compression string `json:"compression"`
	Query       string `json:"query"`

	CSVAllowQuotedRecordDelimiter bool   `json:"csv_allow_quoted_record_delimiter"`
	CSVComments                   string `json:"csv_comments"`
	CSVFieldDelimiter             string `json:"csv_field_delimiter"`
	CSVFileHeaderInfo             string `json:"csv_file_header_info"`
	CSVQuoteCharacter             string `json:"csv_quote_character"`
	CSVQuoteEscapeCharacter       string `json:"csv_quote_escape_character"`
	CSVRecordDelimiter            string `json:"csv_record_delimiter"`
	JSONType                      string `json:"json_type"`
}

func (td *S3DataSource) params(query *queryModel) *s3.SelectObjectContentInput {
	params := &s3.SelectObjectContentInput{
		Bucket: aws.String(td.settings.Bucket),
		Key: aws.String(query.Path),
		ExpressionType: aws.String(s3.ExpressionTypeSql),
		Expression: aws.String(query.Query),
		InputSerialization: &s3.InputSerialization{},
		OutputSerialization: &s3.OutputSerialization{
			JSON: &s3.JSONOutput{
				RecordDelimiter: aws.String(","),
			},
		},
	}

	switch query.Format {
	case "CSV":
		params.InputSerialization.CSV = &s3.CSVInput{}
		params.InputSerialization.CSV.AllowQuotedRecordDelimiter = aws.Bool(query.CSVAllowQuotedRecordDelimiter)
		params.InputSerialization.CSV.FileHeaderInfo = aws.String(query.CSVFileHeaderInfo)

		if query.CSVComments != "" {
			params.InputSerialization.CSV.Comments = aws.String(query.CSVComments)
		}
		if query.CSVFieldDelimiter != "" {
			params.InputSerialization.CSV.FieldDelimiter = aws.String(query.CSVFieldDelimiter)
		}
		if query.CSVQuoteCharacter != "" {
			params.InputSerialization.CSV.QuoteCharacter = aws.String(query.CSVQuoteCharacter)
		}
		if query.CSVQuoteEscapeCharacter != "" {
			params.InputSerialization.CSV.QuoteEscapeCharacter = aws.String(query.CSVQuoteEscapeCharacter)
		}
		if query.CSVRecordDelimiter != "" {
			params.InputSerialization.CSV.RecordDelimiter = aws.String(query.CSVRecordDelimiter)
		}
		break
	case "JSON":
		params.InputSerialization.JSON = &s3.JSONInput{}
		if query.JSONType != "" {
			params.InputSerialization.JSON.Type = aws.String(query.JSONType)
		}
		break
	}

	return params
}

func guessTimeFormat(series []*string, monthfirst bool) (string, error) {
	timeformat := ""
	for _, row := range series {
		if *row == "" {
			continue
		}
		_timeformat, err := dateparse.ParseFormat(*row, monthfirst)
		if err != nil {
			return "", err
		}
		if timeformat == "" {
			timeformat = _timeformat
		} else if timeformat != _timeformat {
			return "", fmt.Errorf("inconsistent time format!")
		}
	}

	if timeformat == "" {
		return "", fmt.Errorf("could not guess timeformat!")
	}

	return timeformat, nil
}

func parseTimeColumn(series []*string, monthfirst bool) *[]time.Time {
	_series := []time.Time{}
	for _, row := range series {
		if *row == "" {
			_series = append(_series, time.Time{})
		}
		_series = append(_series, dateparse.MustParse(*row, monthfirst))
	}

	return &_series
}

func isTimeColumn(series []*string) (bool, *[]time.Time) {
	_, err := guessTimeFormat(series, false)
	if err == nil {
		return true, parseTimeColumn(series, false)
	} else {
		_, err = guessTimeFormat(series, true)
		if err == nil {
			return true, parseTimeColumn(series, true)
		}
	}

	return false, nil
}

func (td *S3DataSource) s3Select(ctx context.Context, params *s3.SelectObjectContentInput) (*data.Frame, error) {
	resp, err := td.svc.SelectObjectContentWithContext(ctx, params)
	if err != nil {
		return nil, err
	}
	defer resp.EventStream.Close()

	payload := []byte{'['}
	for event := range resp.EventStream.Events() {
		switch e := event.(type) {
		case *s3.RecordsEvent:
			payload = append(payload, e.Payload...)
		case *s3.StatsEvent:
			// TODO: report these to user as query cost
			log.DefaultLogger.Info("S3Select", "stats", *e.Details)
		}
	}

	if err := resp.EventStream.Err(); err != nil {
		// TODO: FIXME
		//if aerr, ok := response.Error.(awserr.Error); ok {
		//	switch aerr.Code() {
		//	// Check against specific error codes that need custom handling
		//	}
		//}
		return nil, err
	}

	payload = append(payload[:len(payload) - 1], ']')
	df_wo_types := qframe.ReadJSON(bytes.NewReader(payload))

	// The following hack guesses parameter types
	reader, writer := io.Pipe()
	defer reader.Close()
	go func() {
		df_wo_types.ToCSV(writer)
		writer.Close()
	}()
	df := qframe.ReadCSV(reader)

	// create data frame response
	frame := data.NewFrame("response")

	for column, datatype := range df.ColumnTypeMap() {
		switch datatype {
		case "float":
			frame.Fields = append(frame.Fields,
		                data.NewField(column, nil, df.MustFloatView(column).Slice()),
		        )
		case "string":
			seriesString := df.MustStringView(column).Slice()

			isTime, seriesTime := isTimeColumn(seriesString)

			if isTime {
				frame.Fields = append(frame.Fields,
			                data.NewField(column, nil, *seriesTime),
			        )
			} else {
				frame.Fields = append(frame.Fields,
			                data.NewField(column, nil, seriesString),
			        )
			}
		}
	}

	return frame, nil
}

func (td *S3DataSource) query(ctx context.Context, query backend.DataQuery) backend.DataResponse {
	var qm queryModel
	var frame *data.Frame
	response := backend.DataResponse{}

	response.Error = json.Unmarshal(query.JSON, &qm)
	if response.Error != nil {
		return response
	}

	params := td.params(&qm)
	frame, response.Error = td.s3Select(ctx, params)
	if response.Error != nil {
		return response
	}

	response.Frames = append(response.Frames, frame)

	return response
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (td *S3DataSource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	// TODO: Implement Health Check

	var status = backend.HealthStatusOk
	var message = "Data source is working"

	return &backend.CheckHealthResult{
		Status:  status,
		Message: message,
	}, nil
}

type instanceSettings struct {
	httpClient *http.Client
}

func newDataSourceInstance(setting backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return &instanceSettings{
		httpClient: &http.Client{},
	}, nil
}

func (s *instanceSettings) Dispose() {
	// Called before creatinga a new instance to allow plugin authors
	// to cleanup.
}
