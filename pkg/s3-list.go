package main

import (
	"context"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

func s3List(ctx context.Context, svc *s3.S3, query *Query) (*data.Frame, error) {
	result, err := svc.ListObjectsV2WithContext(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(query.Bucket),
		Prefix: aws.String(query.Path),
		Delimiter: aws.String("/"),
	})
	if err != nil {
		return nil, err
	}

	folders := make([]string, 0)
	formatted := strings.Contains(query.Query, "FORMATTED")

	name := make([]string, 0)
	modified := make([]*time.Time, 0)
	size := make([]*int64, 0)
	for _, prefix := range result.CommonPrefixes {
		parts := strings.Split(*prefix.Prefix, "/")
		part := parts[len(parts) - 2]
		folders = append(folders, part)
		if formatted {
			part = part + ",type=folder,key=" + *prefix.Prefix
		}
		name = append(name, part)
		modified = append(modified, nil)
		size = append(size, nil)
	}
	for _, object := range result.Contents {
		parts := strings.Split(*object.Key, "/")
		part := parts[len(parts) - 1]
		if formatted {
			part = part + ",type=file,key=" + *object.Key
		}
		name = append(name, part)
		modified = append(modified, object.LastModified)
		size = append(size, object.Size)
	}

	frame := data.NewFrame("response")
	frame.Fields = append(frame.Fields, data.NewField("Name", nil, name))
	frame.Fields = append(frame.Fields, data.NewField("Last Modified", nil, modified))
	frame.Fields = append(frame.Fields, data.NewField("Size", nil, size))

	if formatted {
		frame.Fields = append(frame.Fields, data.NewField("Delete", nil, make([]string, len(name))))
		frame.Fields[3].Config = &data.FieldConfig{
			Custom: map[string]interface{}{"width": 100},
		}
	}

	frame.Fields[1].Config = &data.FieldConfig{
		Custom: map[string]interface{}{"width": 200},
		Unit:   "time:YYYY-MM-DD HH:mm:ss",
	}
	frame.Fields[2].Config = &data.FieldConfig{
		Custom: map[string]interface{}{"width": 100, "align": "left"},
		Unit:  "bytes",
	}

	frame.Meta = &data.FrameMeta{
		Custom: map[string]interface{}{"folders": folders},
	}

	return frame, nil
}
