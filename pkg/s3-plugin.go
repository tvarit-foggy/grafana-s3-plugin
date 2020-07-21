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
	JSONTimeField                 string `json:"json_time_field"`
	JSONTimeMonthFirst            bool   `json:"json_time_month_first"`
	JSONTimeBucket                int64  `json:"json_time_bucket"`
}

func (td *S3DataSource) params(query *queryModel) *s3.SelectObjectContentInput {
	params := &s3.SelectObjectContentInput{
		Bucket: aws.String(td.settings.Bucket),
		Key: aws.String(query.Path),
		ExpressionType: aws.String(s3.ExpressionTypeSql),
		Expression: aws.String(query.Query),
		InputSerialization: &s3.InputSerialization{
			CompressionType: aws.String(query.Compression),
                },
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

func (td *S3DataSource) json_time_params(query *queryModel) *s3.SelectObjectContentInput {
	if query.Format != "JSON" || query.JSONTimeField == "" || query.JSONTimeBucket <= 0 {
		return nil
	}

	params := &s3.SelectObjectContentInput{
		Bucket: aws.String(td.settings.Bucket),
		Key: aws.String(query.Path),
		ExpressionType: aws.String(s3.ExpressionTypeSql),
		Expression: aws.String(query.JSONTimeField),
		InputSerialization: &s3.InputSerialization{
			JSON: &s3.JSONInput{
				Type: aws.String(query.JSONType),
			},
			CompressionType: aws.String(query.Compression),
                },
		OutputSerialization: &s3.OutputSerialization{
			JSON: &s3.JSONOutput{
				RecordDelimiter: aws.String(","),
			},
		},
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

func (td *S3DataSource) _s3Select(ctx context.Context, params *s3.SelectObjectContentInput) (*qframe.QFrame, error) {
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
	df := qframe.ReadJSON(bytes.NewReader(payload))

	return &df, nil
}

func (td *S3DataSource) s3Select(ctx context.Context, params *s3.SelectObjectContentInput) (*data.Frame, error) {
	df_wo_types, err := td._s3Select(ctx, params)
	if err != nil {
		return nil, err
	}

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
		case "int":
			view, err := df.IntView(column)
			if err != nil {
				return nil, err
			}
			iseries := view.Slice()
			series := make([]int64, len(iseries))
			for i, v := range iseries {
				series[i] = int64(v)
			}
			frame.Fields = append(frame.Fields,
		                data.NewField(column, nil, series),
		        )
		case "float":
			view, err := df.FloatView(column)
			if err != nil {
				return nil, err
			}
			frame.Fields = append(frame.Fields,
		                data.NewField(column, nil, view.Slice()),
		        )
		case "string":
			view, err := df.StringView(column)
			if err != nil {
				return nil, err
			}

			isTime, series := isTimeColumn(view.Slice())

			if isTime {
				frame.Fields = append(frame.Fields,
			                data.NewField(column, nil, *series),
			        )
			} else {
				frame.Fields = append(frame.Fields,
			                data.NewField(column, nil, view.Slice()),
			        )
			}
		}
	}

	return frame, nil
}

func (td *S3DataSource) s3SelectTime(ctx context.Context, params *s3.SelectObjectContentInput, query *queryModel, frame *data.Frame) error {
	df_wo_types, err := td._s3Select(ctx, params)
	if err != nil {
		return err
	}

	if df_wo_types.Len() == 0 {
		return fmt.Errorf("Unable to fetch time field!")
	}

	column := df_wo_types.ColumnNames()[0]
	view, err := df_wo_types.StringView(column)
	if err != nil {
		return err
	}

	timestamp, err := dateparse.ParseAny(*view.ItemAt(0), query.JSONTimeMonthFirst)
	if err != nil {
		return err
	}

	series := make([]time.Time, 0)
	for i := 0; i < frame.Rows(); i++ {
		series = append(series, timestamp.Add(time.Duration(int64(i) * query.JSONTimeBucket) * time.Nanosecond))
	}

	if column == "_1" {
		column = "time"
	}

	frame.Fields = append(frame.Fields,
                data.NewField(column, nil, series),
        )

	return nil
}

func (td *S3DataSource) query(ctx context.Context, query backend.DataQuery) backend.DataResponse {
	var qm queryModel
	var frame *data.Frame
	response := backend.DataResponse{}

	response.Error = json.Unmarshal(query.JSON, &qm)
	if response.Error != nil {
		return response
	}

	// TODO: FIXME
	//s, err := td.svc.ListObjectsV2WithContext(ctx, &s3.ListObjectsV2Input{
        //        Bucket: aws.String(td.settings.Bucket),
        //        Prefix: aws.String("PlantA/Ma"),
	//	Delimiter: aws.String("/"),
	//})
	//log.DefaultLogger.Info("S3Select", "1111", err)
	//for _, k := range s.CommonPrefixes {
	//	log.DefaultLogger.Info("S3Select", "2222", k.Prefix)
	//}
	//for _, k := range s.Contents {
	//	log.DefaultLogger.Info("S3Select", "3333", k.Key)
	//}

	params := td.params(&qm)
	frame, response.Error = td.s3Select(ctx, params)
	if response.Error != nil {
		return response
	}

	json_time_params := td.json_time_params(&qm)
	if frame.Rows() > 0 && json_time_params != nil {
		response.Error = td.s3SelectTime(ctx, json_time_params, &qm, frame)
		if response.Error != nil {
			return response
		}
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
