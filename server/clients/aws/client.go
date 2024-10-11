package s3client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go/aws/awserr"
	s4 "github.com/aws/aws-sdk-go/service/s3"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"ndb/server/config"
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
	err := s.checkIdObjectExists(ctx, key)
	if err != nil {
		return nil, err
	}

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

func (s *Client) checkIdObjectExists(ctx context.Context, key string) error {
	_, err := s.baseClient.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		var aerr awserr.Error
		if errors.As(err, &aerr) {
			switch aerr.Code() {
			case s4.ErrCodeNoSuchKey:
				_, err = s.baseClient.PutObject(ctx, &s3.PutObjectInput{
					Bucket: aws.String(s.bucket),
					Key:    aws.String(key),
					Body:   io.Reader(bytes.NewBuffer(nil)),
				})
				if err != nil {
					s.log.ErrorContext(
						ctx,
						"couldn't upload new empty file",
						slog.Any("bucket", s.bucket),
						slog.Any("key", key),
						slog.Any("error", err),
					)
					return err
				}
			default:
				s.log.ErrorContext(
					ctx,
					"couldn't get object",
					slog.Any("bucket", s.bucket),
					slog.Any("key", key),
					slog.Any("error", err),
				)
				return err

			}
		}
	}
	return nil
}

func (s *Client) UploadFile(ctx context.Context, reader io.Reader, url string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, url, reader)
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	return nil
}

func (s *Client) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	output, err := s.baseClient.GetObject(ctx,
		&s3.GetObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
		},
	)

	if err != nil {
		return nil, err
	}

	return output.Body, nil
}
