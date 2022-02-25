/*
Copyright © 2021 David Morgan <dmorgan81@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package s3

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/dmorgan81/buzzel/pkg/cache"
	health "github.com/etherlabsio/healthcheck/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Cache struct {
	bucket  string
	client  *s3.Client
	uploads chan *upload
}

type upload struct {
	ctx context.Context
	in  *s3.PutObjectInput
}

var _ cache.Cache = &Cache{}

func NewCache(bucket string) (*Cache, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	if log.Logger.GetLevel() == zerolog.DebugLevel {
		cfg.ClientLogMode = aws.LogRequest | aws.LogResponse
	}

	client := s3.NewFromConfig(cfg)
	if _, err := client.HeadBucket(context.TODO(), &s3.HeadBucketInput{Bucket: aws.String(bucket)}); err != nil {
		return nil, err
	}

	uploads := make(chan *upload)
	go func() {
		uploader := manager.NewUploader(client)
		for upload := range uploads {
			// we don't want to use the upload ctx because the actual
			// upload to S3 is asynchronous from the client's POV
			if _, err := uploader.Upload(context.TODO(), upload.in); err != nil {
				log := zerolog.Ctx(upload.ctx).With().Caller().Logger()
				log.Err(err).Send()
			}
		}
	}()

	return &Cache{bucket, client, uploads}, nil
}

func resolve(store cache.Store, key cache.Key) string {
	return path.Join(string(store), string(key))
}

func (c *Cache) Exists(ctx context.Context, store cache.Store, key cache.Key) error {
	path := resolve(store, key)
	log := zerolog.Ctx(ctx).With().Caller().Logger()
	log.Debug().Str("path", path).Send()

	if _, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(path),
	}); err != nil {
		var awserr *awshttp.ResponseError
		if errors.As(err, &awserr) && awserr.HTTPStatusCode() == http.StatusNotFound {
			return cache.ErrNotFound
		}
		return err
	}
	return nil
}

func (c *Cache) Reader(ctx context.Context, store cache.Store, key cache.Key) (io.Reader, int64, error) {
	path := resolve(store, key)
	log := zerolog.Ctx(ctx).With().Caller().Logger()
	log.Debug().Str("path", path).Send()

	out, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, -1, cache.ErrNotFound
		}
		return nil, -1, err
	}

	return out.Body, out.ContentLength, nil
}

func (c *Cache) Writer(ctx context.Context, store cache.Store, key cache.Key) (io.Writer, error) {
	path := resolve(store, key)
	log := zerolog.Ctx(ctx).With().Caller().Logger()
	log.Debug().Str("path", path).Send()

	pr, pw := io.Pipe()
	c.uploads <- &upload{
		ctx: ctx,
		in: &s3.PutObjectInput{
			Bucket:      aws.String(c.bucket),
			Key:         aws.String(path),
			ContentType: aws.String("application/octect-stream"),
			Body:        pr,
		},
	}
	return pw, nil
}

var _ health.Checker = &Cache{}

func (c *Cache) Check(ctx context.Context) error {
	_, err := c.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(c.bucket)})
	return err
}
