package main

import (
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
)

type S3DataSource struct {
	im instancemgmt.InstanceManager

	s3       *s3.S3
	sts      *sts.STS
	settings struct {
		Bucket    string `json:"bucket" binding:"Required"`
		Region    string `json:"region" binding:"Required"`
		AccessKey string `json:"accessKey"`
	}
}

type Query struct {
	Bucket      string `json:"-"`
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

type instanceSettings struct {
	httpClient *http.Client
}
