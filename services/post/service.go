package post

import (
	"context"
	"image"
	"log/slog"
	"mime/multipart"

	"github.com/google/uuid"

	"ndb/app/models"
	awsS3 "ndb/clients/aws"
	"ndb/repositories/posts"
	"ndb/repositories/posts/model"
)

type Service struct {
	s3Client *awsS3.Client
	store    *posts.Store
	log      *slog.Logger
}

func NewService(s3Client *awsS3.Client, store *posts.Store, log *slog.Logger) *Service {
	return &Service{
		s3Client: s3Client,
		store:    store,
		log:      log,
	}
}

func (s *Service) CreatePost(ctx context.Context, file multipart.File, data *models.CreatePostRequest) (string, error) {
	imageDoc, _, err := image.Decode(file)
	if err != nil {
		s.log.ErrorContext(ctx, "Error encoding image", slog.Any("error", err))
		return "", err
	}

	post := model.PostFrom(data)
	post.ImageName = uuid.New().String()

	_, err = s.store.CreatePost(ctx, post, data.Thread)
	if err != nil {
		s.log.ErrorContext(ctx, "Error inserting post", slog.Any("error", err))
		return "", err

	}

	presignedUrl, err := s.s3Client.UploadPresignURL(ctx, post.ImageName)
	if err != nil {
		return "", err
	}

	err = s.s3Client.UploadFile(ctx, imageDoc, presignedUrl.URL)
	if err != nil {
		return "", err
	}

	return post.Title, nil
}
