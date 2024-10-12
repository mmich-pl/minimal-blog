package file

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"io"
	"log/slog"
	awsS3 "ndb/server/clients/aws"
	"ndb/server/config"
	"time"
)

type Service struct {
	s3Client *awsS3.Client
	log      *slog.Logger
}

func NewService(s3Client *awsS3.Client, log *slog.Logger) *Service {
	return &Service{
		s3Client: s3Client,
		log:      log,
	}
}

func (s *Service) InsertFile(
	ctx context.Context,
	fileName string,
	file []byte,
) error {
	// Generate a presigned URL for the markdown file
	presignedUrl, err := s.s3Client.UploadPresignURL(ctx, fileName)
	if err != nil {
		return err
	}

	// Upload the markdown file with the header to S3
	err = s.s3Client.UploadFile(ctx, bytes.NewReader(file), presignedUrl.URL)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) GetFile(
	ctx context.Context,
	contentFile string,
) (io.ReadCloser, error) {
	readCloser, err := s.s3Client.Get(ctx, contentFile)
	if err != nil {
		s.log.ErrorContext(
			ctx,
			"Could not retrieve file from S3",
			slog.Any("error", err),
			slog.Any("image_name", contentFile),
		)
		return nil, err
	}

	return readCloser, nil
}

type CachedService struct {
	redisClient *redis.Client
	base        *Service
	log         *slog.Logger
	ttl         time.Duration
}

func NewCachedService(s3Client *awsS3.Client, cfg *config.Redis, log *slog.Logger) *CachedService {
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.Address,
	})

	return &CachedService{
		base:        NewService(s3Client, log),
		redisClient: redisClient,
		log:         log,
		ttl:         cfg.TTL,
	}
}

func (c *CachedService) InsertFile(
	ctx context.Context,
	fileName string,
	file []byte,
) error {
	err := c.base.InsertFile(ctx, fileName, file)
	if err != nil {
		return err
	}

	err = c.set(ctx, fileName, file)
	if err != nil {
		return err
	}

	return nil
}

func (c *CachedService) GetFile(ctx context.Context, fileName string) (io.ReadCloser, error) {
	val, err := c.redisClient.Get(ctx, fileName).Bytes()
	if errors.Is(err, redis.Nil) {
		c.log.InfoContext(
			ctx,
			"File not found in redis cache",
			slog.Any("file_name", fileName),
		)

		var rc io.ReadCloser
		rc, err = c.base.GetFile(ctx, fileName)
		if err != nil {
			return nil, err
		}

		var buf bytes.Buffer
		reader := io.TeeReader(rc, &buf)
		err = c.set(ctx, fileName, buf.Bytes())
		if err != nil {
			c.log.ErrorContext(
				ctx,
				"Failed to copy reader",
				slog.Any("err", err))
			return nil, err
		}

		return io.NopCloser(reader), nil
	}
	if err != nil {
		c.log.ErrorContext(
			ctx,
			"Failed to retrieve file from redis cache",
			slog.Any("error", err),
			slog.Any("file_name", fileName),
		)
		return nil, fmt.Errorf("failed to retrieve file from redis cache: %v", err)
	}

	c.log.InfoContext(
		ctx,
		"Retrieved file from redis cache",
		slog.Any("file_name", fileName),
	)
	return io.NopCloser(bytes.NewReader(val)), nil
}

func (c *CachedService) set(ctx context.Context, fileName string, file []byte) error {
	err := c.redisClient.Set(ctx, fileName, file, c.ttl).Err()
	if err != nil {
		c.log.ErrorContext(
			ctx,
			"Failed to set redis cache",
			slog.Any("error", err),
			slog.Any("file_name", fileName),
		)
		return fmt.Errorf("failed to set redis cache: %v", err)
	}
	return nil
}
