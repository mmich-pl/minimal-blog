package s3client

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"ndb/config"
)

const presignTTL = 5 * time.Minute

type Client struct {
	baseClient    *s3.Client
	presignClient *s3.PresignClient
	log           *slog.Logger
	bucket        string
}

func New(
	ctx context.Context,
	logger *slog.Logger,
	cfg *config.S3,
) (*Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.Key, cfg.Secret, ""),
		),
	)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to load aws configuration", slog.Any("error", err))
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(fmt.Sprintf("%s:%d", cfg.BaseUrl, cfg.Port))
	})

	return &Client{
		baseClient:    client,
		presignClient: s3.NewPresignClient(client),
		log:           logger,
		bucket:        cfg.Bucket,
	}, nil
}

func (s *Client) UploadPresignURL(ctx context.Context, key string) (*v4.PresignedHTTPRequest, error) {
	presignedUrl, err := s.presignClient.PresignPutObject(
		ctx,
		&s3.PutObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
		},
		s3.WithPresignExpires(time.Minute*15),
	)
	if err != nil {
		s.log.ErrorContext(ctx,
			"Couldn't get a presigned URL\n",
			slog.Any("key", key),
			slog.Any("bucket", s.bucket),
			slog.Any("error", err),
		)
		return nil, err
	}
	s.log.InfoContext(
		ctx,
		"Generated presigned URL",
		slog.Any("key", key),
		slog.Any("ttl", presignTTL),
		slog.Any("bucket", s.bucket),
	)
	return presignedUrl, nil
}

func (s *Client) UploadFile(ctx context.Context, file image.Image, url string) error {
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, file, nil)
	if err != nil {
		return nil
	}
	body := io.Reader(&buf)
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, url, body)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "image/jpeg")

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		s.log.ErrorContext(ctx, "Error sending upload request", slog.Any("error", err))
		return err
	}
	defer resp.Body.Close()
	return err
}
