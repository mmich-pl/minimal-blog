package posts

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"image"
	"io"
	"log/slog"
	"mime/multipart"

	"ndb/app/models"
	awsS3 "ndb/clients/aws"
	"ndb/repositories/posts"
	"ndb/repositories/posts/model"
)

var ErrNotFound = errors.New("not found")

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

func (s *Service) CreateThread(ctx context.Context, data *models.CreateThreadRequest) (string, error) {
	thread := model.ThreadFrom(data)
	threadID, err := s.store.CreateThread(ctx, thread)
	if err != nil {
		s.log.ErrorContext(ctx, "Error creating thread", slog.Any("error", err))
		return "", err
	}

	return threadID, nil
}

func (s *Service) CreatePost(ctx context.Context, file multipart.File, data *models.CreatePostRequest) (string, error) {
	imageDoc, _, err := image.Decode(file)
	if err != nil {
		s.log.ErrorContext(ctx, "Error encoding image", slog.Any("error", err))
		return "", err
	}

	post := model.PostFrom(data)
	post.ImageName = fmt.Sprintf("%s.jpg", uuid.New().String())

	post.PostID, err = s.store.CreatePost(ctx, post, data.Thread)
	if err != nil {
		s.log.ErrorContext(ctx, "Error creating post", slog.Any("error", err))
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

	return post.PostID, nil
}

func (s *Service) GetPost(ctx context.Context, postID string) (*models.Post, io.ReadCloser, error) {
	post, err := s.store.GetPost(ctx, postID)
	if err != nil {
		s.log.ErrorContext(
			ctx,
			"Error getting posts",
			slog.Any("error", err),
			slog.Any("post_id", post),
		)
		return nil, nil, err
	}
	post.ViewCount += 1

	readCloser, err := s.s3Client.Get(ctx, post.ImageName)
	if err != nil {
		s.log.ErrorContext(
			ctx,
			"Could not retrieve file",
			slog.Any("error", err),
			slog.Any("image_name", post.ImageName),
		)
		return nil, nil, err
	}

	return &models.Post{
		PostID:    post.PostID,
		UserID:    post.UserID,
		Title:     post.Title,
		Content:   post.Content,
		ViewCount: post.ViewCount,
		ImageName: post.ImageName,
	}, readCloser, nil
}

func (s *Service) ListThreads(ctx context.Context) ([]*models.Thread, error) {
	t, err := s.store.ListThreads(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, "Error listing threads", slog.Any("error", err))
		return nil, err
	}

	if len(t) == 0 {
		return nil, fmt.Errorf("%w: no threads was found", ErrNotFound)
	}

	var threads []*models.Thread
	for _, thread := range t {
		threads = append(threads, &models.Thread{
			ThreadID: thread.ThreadID,
			Name:     thread.Name,
			Tags:     thread.Tags,
		})
	}

	return threads, nil
}

func (s *Service) ListTags(ctx context.Context) ([]string, error) {
	tags, err := s.store.ListTags(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, "Error listing threads", slog.Any("error", err))
		return nil, err
	}

	if len(tags) == 0 {
		return nil, fmt.Errorf("%w: no tags was found", ErrNotFound)
	}

	return tags, nil
}
