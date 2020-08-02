package main

import (
	"context"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/data"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

func s3Delete(ctx context.Context, svc *s3.S3, query *Query) (*data.Frame, error) {
	folder := strings.Contains(query.Query, "folder")

	if (folder) {
	        objects, err := svc.ListObjectsV2WithContext(ctx, &s3.ListObjectsV2Input{
        	        Bucket: aws.String(query.Bucket),
                	Prefix: aws.String(query.Path),
	                Delimiter: aws.String("/"),
        	})
	        if err != nil {
        	        return nil, err
	        }

		ids := make([]*s3.ObjectIdentifier, 0)
		for _, object := range objects.Contents {
			ids = append(ids, &s3.ObjectIdentifier{Key: object.Key})
		}

		_, err = svc.DeleteObjectsWithContext(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(query.Bucket),
			Delete: &s3.Delete{
				Objects: ids,
			},
		})
		if err != nil {
			return nil, err
		}
	} else {
		_, err := svc.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(query.Bucket),
			Key: aws.String(query.Path),
		})
		if err != nil {
			return nil, err
		}
	}

	return data.NewFrame("response"), nil
}
