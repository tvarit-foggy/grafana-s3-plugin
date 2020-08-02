package main

import (
	"context"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
)


func stsSession(ctx context.Context, svc *sts.STS, query *Query) (*data.Frame, error) {
	result, err := svc.GetSessionTokenWithContext(ctx, &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int64(900),
	})
	if err != nil {
		return nil, err
	}

	frame := data.NewFrame("response")
	frame.Fields = append(frame.Fields, data.NewField("AccessKeyId", nil, []*string{result.Credentials.AccessKeyId}))
	frame.Fields = append(frame.Fields, data.NewField("SecretAccessKey", nil, []*string{result.Credentials.SecretAccessKey}))
	frame.Fields = append(frame.Fields, data.NewField("SessionToken", nil, []*string{result.Credentials.SessionToken}))
	frame.Fields = append(frame.Fields, data.NewField("Expiration", nil, []*time.Time{result.Credentials.Expiration}))

	return frame, nil
}
