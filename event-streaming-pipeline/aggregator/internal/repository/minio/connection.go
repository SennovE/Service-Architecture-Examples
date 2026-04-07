package minio

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func Connect(host string, port int, user string, password string, bucket string) (*minio.Client, error) {
	client, err := minio.New(
		fmt.Sprintf("%s:%d", host, port),
		&minio.Options{
			Creds: credentials.NewStaticV4(user, password, ""),
		},
	)
	ctx := context.Background()
	err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
	if err != nil {
		exists, errBucketExists := client.BucketExists(ctx, bucket)
		if errBucketExists != nil || !exists {
			return nil, err
		}
	}
	return client, nil
}
