package posts

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log/slog"
	"mime/multipart"

	apimodel "ndb/server/app/models"
	"ndb/server/repositories/posts"
	"ndb/server/repositories/posts/model"
)

var ErrNotFound = errors.New("not found")

type FileService interface {
	InsertFile(
		ctx context.Context,
		fileName string,
		file []byte,
	) error
	GetFile(
		ctx context.Context,
		fileName string,
	) (io.ReadCloser, error)
}

type Service struct {
	store       *posts.Store
	log         *slog.Logger
	fileManager FileService
}

func NewService(fileManager FileService, store *posts.Store, log *slog.Logger) *Service {
	return &Service{
		fileManager: fileManager,
		store:       store,
		log:         log,
	}
}

func (s *Service) CreateThread(ctx context.Context, data *apimodel.CreateThreadRequest) (string, error) {
	thread := model.ThreadFrom(data)
	threadID, err := s.store.CreateThread(ctx, thread)
	if err != nil {
		s.log.ErrorContext(ctx, "Error creating thread", slog.Any("error", err))
		return "", err
	}

	return threadID, nil
}

func (s *Service) CreatePost(ctx context.Context, file multipart.File, data *apimodel.CreatePostRequest) (string, error) {
	// Create the Post object from the request data
	post := model.PostFrom(data)
	post.ContentFile = fmt.Sprintf("%s.md", uuid.New().String())

	// Set the post ID and store the post metadata
	var err error
	post.PostID, err = s.store.CreatePost(ctx, post, data.Thread)
	if err != nil {
		s.log.ErrorContext(ctx, "Error creating post", slog.Any("error", err))
		return "", err
	}

	// Prepare the markdown header
	header := fmt.Sprintf(
		"---\ntitle: \"%s\"\nthread: \"%s\"\ndate: \"%s\"\n---\n\n",
		post.Title, post.ThreadID, post.UpdatedAt,
	)

	// Read the content of the provided markdown file
	contentBytes, err := io.ReadAll(file)
	if err != nil {
		s.log.ErrorContext(ctx, "Error reading file content", slog.Any("error", err))
		return "", err
	}

	if len(contentBytes) == 0 {
		return "", errors.New("tried to upload empty image")
	}

	contentWithHeader := append([]byte(header), contentBytes...)

	err = s.fileManager.InsertFile(ctx, post.ContentFile, contentWithHeader)
	if err != nil {
		s.log.ErrorContext(ctx, "Error inserting file", slog.Any("error", err))
		return "", err
	}

	return post.PostID, nil
}

func (s *Service) GetPostMetadata(ctx context.Context, postID string) (*apimodel.Post, error) {
	post, err := s.store.GetPost(ctx, postID)
	if err != nil {
		s.log.ErrorContext(
			ctx,
			"Error getting posts",
			slog.Any("error", err),
			slog.Any("post_id", postID),
		)
		return nil, err
	}
	post.ViewCount += 1

	return &apimodel.Post{
		PostID:     post.PostID,
		UserID:     post.UserID,
		ThreadID:   post.ThreadID,
		Title:      post.Title,
		ViewCount:  post.ViewCount,
		ContentFle: post.ContentFile,
	}, nil
}

func (s *Service) GetPostMarkdown(ctx context.Context, contentFile string) (io.ReadCloser, error) {
	return s.fileManager.GetFile(ctx, contentFile)
}

func (s *Service) GetPostsWithLimit(ctx context.Context, limit int) (map[string][]*apimodel.Post, error) {
	posts, err := s.store.GetPostsWithLimit(ctx, limit)
	if err != nil {
		s.log.ErrorContext(
			ctx,
			"Error getting posts with limit",
			slog.Any("error", err),
			slog.Any("limit", limit),
		)
		return nil, err
	}

	postsResp := make(map[string][]*apimodel.Post, len(posts)) // Initialize the map
	for key, items := range posts {
		for _, p := range items {
			// Check if the key already exists in the map
			if _, exists := postsResp[key]; !exists {
				// If not, initialize an empty array for this key
				postsResp[key] = []*apimodel.Post{}
			}

			// Append the mapped post to the slice
			postsResp[key] = append(postsResp[key], &apimodel.Post{
				PostID:     p.PostID,
				UserID:     p.UserID,
				Title:      p.Title,
				ThreadID:   p.ThreadID,
				ViewCount:  p.ViewCount,
				ContentFle: p.ContentFile,
			})
		}
	}

	return postsResp, nil
}

func (s *Service) ListThreads(ctx context.Context) ([]*apimodel.Thread, error) {
	t, err := s.store.ListThreads(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, "Error listing threads", slog.Any("error", err))
		return nil, err
	}

	if len(t) == 0 {
		return nil, fmt.Errorf("%w: no threads was found", ErrNotFound)
	}

	var threads []*apimodel.Thread
	for _, thread := range t {
		threads = append(threads, &apimodel.Thread{
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
