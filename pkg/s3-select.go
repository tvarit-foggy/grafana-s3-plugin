package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/KamalGalrani/dateparse"
	"github.com/tobgu/qframe"
)

// Gets S3 Select params for data query
func get_s3_select_query_params(query *Query) *s3.SelectObjectContentInput {
	params := &s3.SelectObjectContentInput{
		Bucket: aws.String(query.Bucket),
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
	case "JSON":
		params.InputSerialization.JSON = &s3.JSONInput{}
		if query.JSONType != "" {
			params.InputSerialization.JSON.Type = aws.String(query.JSONType)
		}
	}

	return params
}

// Gets S3 Select params for time field query. Only applicable to JSON
func get_s3_select_time_params(query *Query) *s3.SelectObjectContentInput {
	if query.Format != "JSON" || query.JSONTimeField == "" || query.JSONTimeBucket <= 0 {
		return nil
	}

	params := &s3.SelectObjectContentInput{
		Bucket: aws.String(query.Bucket),
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

// guess time format for a column
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

// parse string column into time
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

// check if column is time column
func isTimeColumn(series []*string) (bool, *[]time.Time) {
	// TODO: merge guess and parse to reduce overhead
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


func s3SelectBase(ctx context.Context, svc *s3.S3, params *s3.SelectObjectContentInput) (*qframe.QFrame, error) {
	resp, err := svc.SelectObjectContentWithContext(ctx, params)
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
		return nil, err
	}

	payload = append(payload[:len(payload) - 1], ']')
	df := qframe.ReadJSON(bytes.NewReader(payload))

	return &df, nil
}

func s3SelectQuery(ctx context.Context, svc *s3.S3, params *s3.SelectObjectContentInput) (*data.Frame, error) {
	df_wo_types, err := s3SelectBase(ctx, svc, params)
	if err != nil {
		return nil, err
	}

	// The following hack guesses parameter types
	reader, writer := io.Pipe()
	defer reader.Close()
	go func() {
		err := df_wo_types.ToCSV(writer)
		log.DefaultLogger.Error("S3Select", "df.ToCSV", err.Error())
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

func s3SelectTime(ctx context.Context, svc *s3.S3, params *s3.SelectObjectContentInput, query *Query, frame *data.Frame) error {
	df_wo_types, err := s3SelectBase(ctx, svc, params)
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

func s3Select(ctx context.Context, svc *s3.S3, query *Query) (*data.Frame, error) {
	// TODO: Add support for time filter
	// TODO: Add support for reading multipart
	query_params := get_s3_select_query_params(query)
	frame, err := s3SelectQuery(ctx, svc, query_params)
	if err != nil {
		return nil, err
	}

	time_params := get_s3_select_time_params(query)
	if frame.Rows() > 0 && time_params != nil {
		err = s3SelectTime(ctx, svc, time_params, query, frame)
		if err != nil {
			return nil, err
		}
	}

	return frame, nil
}
