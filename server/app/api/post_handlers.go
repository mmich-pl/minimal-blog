package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/render"

	"ndb/server/app/models"
	"ndb/server/errors"
)

// CreatePostHandler handles the creation of a new post along with a markdown file upload.
//
// @Summary Create a new post with a markdown file
// @Description This endpoint allows users to create a new post by submitting text data (title, content, thread, user_id) and a markdown file (.md).
// The markdown file is saved in the user's designated S3 bucket, and the post details are saved in MongoDB.
// @Tags posts
// @Accept multipart/form-data
// @Produce json
// @Param markdown formData file true "Markdown File"
// @Param title formData string true "Title of the post"
// @Param thread formData string true "ID of the thread to which the post belongs"
// @Param user_id formData integer true "ID of the user creating the post"
// @Success 200 {object} models.PostCreationResponse
// @Failure 400 {object} errors.ErrResponse "Bad Request"
// @Failure 500 {object} errors.ErrResponse "Internal Server Error"
// @Router /api/v1/posts [post]
func (s *Server) CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	err := r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		s.log.ErrorContext(ctx, "Unable to parse form", slog.Any("error", err))
		render.Render(w, r, errors.ErrBadRequest)
		return
	}

	form := r.MultipartForm
	if form == nil {
		s.log.ErrorContext(ctx, "No multipart form data found", slog.Any("error", err))
		render.Render(w, r, &errors.ErrResponse{
			Err:            fmt.Errorf("multipart form is nil"),
			HTTPStatusCode: http.StatusBadRequest,
			Message:        "Invalid form data",
		})
		return
	}

	// Validate required fields
	requiredFields := []string{"title", "thread", "user_id"}
	if err = validatePostForm(form.Value, requiredFields...); err != nil {
		s.log.ErrorContext(ctx, "Form validation error", slog.Any("error", err))
		render.Render(w, r, &errors.ErrResponse{
			Err:            err,
			HTTPStatusCode: http.StatusBadRequest,
			Message:        err.Error(),
		})
		return
	}

	// Extract fields from the form
	title := form.Value["title"][0]
	thread := form.Value["thread"][0]
	user := form.Value["user_id"][0]

	userID, err := strconv.Atoi(user)
	if err != nil {
		s.log.ErrorContext(ctx, "Cannot parse userID", slog.Any("error", err))
		render.Render(w, r, errors.ErrInternalServerError)
	}

	// Handle markdown file
	files := form.File["markdown"]
	if len(files) == 0 {
		s.log.ErrorContext(ctx, "No markdown file found in form", slog.Any("error", err))
		render.Render(w, r, &errors.ErrResponse{
			Err:            fmt.Errorf("no markdown file provided"),
			HTTPStatusCode: http.StatusBadRequest,
			Message:        "No markdown file provided",
		})
		return
	}

	mdFile := files[0]
	file, err := mdFile.Open()
	if err != nil {
		s.log.ErrorContext(ctx, "Error opening markdown file", slog.Any("error", err))
		render.Render(w, r, errors.ErrInternalServerError)
		return
	}
	defer file.Close()

	// Process the post creation
	data := models.CreatePostRequest{
		Title:  title,
		Thread: thread,
		UserID: int64(userID),
	}

	postID, err := s.postService.CreatePost(ctx, file, &data)
	if err != nil {
		s.log.ErrorContext(ctx, "Error creating post", slog.Any("error", err))
		render.Render(w, r, errors.ErrInternalServerError)
		return
	}

	// Return response
	render.Render(w, r, &models.PostCreationResponse{
		Status: http.StatusOK,
		PostID: postID,
	})
}

func validatePostForm(form map[string][]string, requiredFields ...string) error {
	for _, field := range requiredFields {
		if len(form[field]) == 0 {
			return fmt.Errorf("missing or empty field: %s", field)
		}
	}
	return nil
}

// GetPostLimitHandler handles the fetching of a post along with an image file.
//
// @Summary Retrieve post data along with the associated image
// @Description Fetch post details from Neo4j along with an image file stored in S3. The response contains post details in JSON format followed by the image file.
// @Tags posts
// @Accept json
// @Produce json
// @Param limit query int true "limit"
// @Header 200 {string} Content-Type "application/json"
// @Success 200 {object} []models.Post "Posts"
// @Failure 400 {object} errors.ErrResponse "Invalid request or post not found"
// @Failure 500 {object} errors.ErrResponse "Internal server error"
// @Router /api/v1/posts/{limit} [get]
func (s *Server) GetPostLimitHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit := r.URL.Query().Get("limit")

	if limit == "" {
		limit = "3"
	}

	l, err := strconv.Atoi(limit)
	if err != nil {
		s.log.ErrorContext(ctx, "Cannot parse limit", slog.Any("error", err))
		render.Render(w, r, errors.ErrBadRequest)
	}

	posts, err := s.postService.GetPostsWithLimit(ctx, l)
	if err != nil {
		s.log.ErrorContext(ctx, "Error getting posts", slog.Any("error", err))
		render.Render(w, r, errors.ErrInternalServerError)
	}

	// Write posts data as JSON
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(posts)
	if err != nil {
		s.log.ErrorContext(ctx, "Error encoding post", slog.Any("error", err))
		render.Render(w, r, errors.ErrInternalServerError)
		return
	}
}

// GetPostMetadataHandler handles the fetching of post metadata.
//
// @Summary Retrieve post metadata
// @Description Fetch post details from Neo4j in JSON format.
// @Tags posts
// @Accept json
// @Produce json
// @Param id query string true "Post ID"
// @Success 200 {object} models.Post "Post metadata"
// @Failure 400 {object} errors.ErrResponse "Invalid request or post not found"
// @Failure 500 {object} errors.ErrResponse "Internal server error"
// @Router /api/v1/posts/{id}/metadata [get]
func (s *Server) GetPostMetadataHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postID := r.URL.Query().Get("id")

	if postID == "" {
		render.Render(w, r, &errors.ErrResponse{
			HTTPStatusCode: http.StatusBadRequest,
			Message:        "postID is empty",
		})
		return
	}

	post, err := s.postService.GetPostMetadata(ctx, postID)
	if err != nil {
		s.log.ErrorContext(ctx, "Error getting post metadata", slog.Any("error", err))
		render.Render(w, r, errors.ErrInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(post); err != nil {
		s.log.ErrorContext(ctx, "Error encoding post metadata", slog.Any("error", err))
		render.Render(w, r, errors.ErrInternalServerError)
	}
}

// GetPostMarkdownHandler handles the fetching of a post markdown file.
//
// @Summary Retrieve post markdown file
// @Description Fetch the markdown file associated with a post from S3.
// @Tags posts
// @Produce text/markdown
// @Param id query string true "Content File ID"
// @Success 200 {file} file "Markdown file"
// @Failure 400 {object} errors.ErrResponse "Invalid request or post not found"
// @Failure 500 {object} errors.ErrResponse "Internal server error"
// @Router /api/v1/posts/{id}/markdown [get]
func (s *Server) GetPostMarkdownHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postID := r.URL.Query().Get("id")

	if postID == "" {
		render.Render(w, r, &errors.ErrResponse{
			HTTPStatusCode: http.StatusBadRequest,
			Message:        "postID is empty",
		})
		return
	}

	file, err := s.postService.GetPostMarkdown(ctx, postID)
	if err != nil {
		s.log.ErrorContext(ctx, "Error getting post markdown", slog.Any("error", err))
		render.Render(w, r, errors.ErrInternalServerError)
		return
	}
	defer file.Close()

	// Set headers for markdown file response
	w.Header().Set("Content-Type", "text/markdown")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s", "post.md"))

	// Stream the markdown file to the response
	if _, err = io.Copy(w, file); err != nil {
		s.log.ErrorContext(ctx, "Error writing markdown to response", slog.Any("error", err))
		render.Render(w, r, errors.ErrInternalServerError)
	}
}
