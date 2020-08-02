package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/data"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
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

func (ds *S3DataSource) authenticate(ctx context.Context, req *backend.QueryDataRequest) error {
	config := aws.Config{
		Region: aws.String(ds.settings.Region),
	}

	if ds.settings.AccessKey != "" {
		if secretKey, found := req.PluginContext.DataSourceInstanceSettings.DecryptedSecureJSONData["secretKey"]; found {
			config.Credentials = credentials.NewStaticCredentials(ds.settings.AccessKey, secretKey, "")
		}
	}

	sess, err := session.NewSession()
	if err != nil {
		return err
	}

	ds.s3 = s3.New(sess, &config)
	ds.sts = sts.New(sess, &config)

	return nil
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifer).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (ds *S3DataSource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {

	// Unmarshal the json into datasource settings
	if err := json.Unmarshal(req.PluginContext.DataSourceInstanceSettings.JSONData, &ds.settings); err != nil {
		return nil, err
	}

	// create response struct
	response := backend.NewQueryDataResponse()

	// authenticate with AWS services
	if err := ds.authenticate(ctx, req); err != nil {
		return nil, err
	}

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		res := ds.query(ctx, q)

		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

func (ds *S3DataSource) query(ctx context.Context, dataQuery backend.DataQuery) backend.DataResponse {
	var query Query
	var frame *data.Frame
	response := backend.DataResponse{}

	response.Error = json.Unmarshal(dataQuery.JSON, &query)
	if response.Error != nil {
		return response
	}
	query.Bucket = ds.settings.Bucket

	if strings.HasPrefix(query.Query, "LIST") {
		frame, response.Error = s3List(ctx, ds.s3, &query)
		if response.Error != nil {
			return response
		}
		response.Frames = append(response.Frames, frame)
	} else if strings.HasPrefix(query.Query, "UPLOAD") {
		frame, response.Error = stsSession(ctx, ds.sts, &query)
		if response.Error != nil {
			return response
		}
		response.Frames = append(response.Frames, frame)
	} else if strings.HasPrefix(query.Query, "DELETE") {
		frame, response.Error = s3Delete(ctx, ds.s3, &query)
		if response.Error != nil {
			return response
		}
		response.Frames = append(response.Frames, frame)
	} else {
		frame, response.Error = s3Select(ctx, ds.s3, &query)
		if response.Error != nil {
			return response
		}

		response.Frames = append(response.Frames, frame)
	}

	return response
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (ds *S3DataSource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	if err := json.Unmarshal(req.PluginContext.DataSourceInstanceSettings.JSONData, &ds.settings); err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: err.Error(),
		}, nil
	}

	config := aws.Config{
		Region: aws.String(ds.settings.Region),
	}

	if ds.settings.AccessKey != "" {
		if secretKey, found := req.PluginContext.DataSourceInstanceSettings.DecryptedSecureJSONData["secretKey"]; found {
			config.Credentials = credentials.NewStaticCredentials(ds.settings.AccessKey, secretKey, "")
		}
	}

	sess, err := session.NewSession()
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: err.Error(),
		}, nil
	}

	ds.s3 = s3.New(sess, &config)

	_, err = ds.s3.ListObjectsV2WithContext(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(ds.settings.Bucket),
	})
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: err.Error(),
		}, nil
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Data source is working",
	}, nil
}

func newDataSourceInstance(setting backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return &instanceSettings{
		httpClient: &http.Client{},
	}, nil
}
