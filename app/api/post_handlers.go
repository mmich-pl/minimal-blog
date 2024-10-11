package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/render"

	"ndb/app/models"
	"ndb/errors"
)

// CreatePostHandler handles the creation of a new posts along with an image upload.
//
// @Summary Create a new posts with an image
// @Description This endpoint allows users to create a new posts by submitting text data (title, content, thread, user_id) and an image file.
// The image is saved in the user's designated S3 bucket, and the posts details are saved in MongoDB.
// @Tags posts
// @Accept multipart/form-data
// @Produce json
// @Param image formData file true "Image File"
// @Param title formData string true "Title of the posts"
// @Param content formData string true "Content of the posts"
// @Param thread formData string true "ID of the thread to which the posts belongs"
// @Param user_id formData integer true "ID of the user creating the posts"
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
	requiredFields := []string{"title", "content", "thread", "user_id"}
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
	content := form.Value["content"][0]

	thread := form.Value["thread"][0]
	user := form.Value["user_id"][0]

	userID, err := strconv.Atoi(user)
	if err != nil {
		s.log.ErrorContext(ctx, "Cannot parse userID", slog.Any("error", err))
		render.Render(w, r, errors.ErrInternalServerError)
	}

	// Handle file
	files := form.File["image"]
	if len(files) == 0 {
		s.log.ErrorContext(ctx, "No image file found in form", slog.Any("error", err))
		render.Render(w, r, &errors.ErrResponse{
			Err:            fmt.Errorf("no image file provided"),
			HTTPStatusCode: http.StatusBadRequest,
			Message:        "No image file provided",
		})
		return
	}

	imageFile := files[0]
	file, err := imageFile.Open()
	if err != nil {
		s.log.ErrorContext(ctx, "Error opening image file", slog.Any("error", err))
		render.Render(w, r, errors.ErrInternalServerError)
		return
	}
	defer file.Close()

	// Process the posts creation (similarly as before)
	data := models.CreatePostRequest{
		Title:   title,
		Content: content,
		Thread:  thread,
		UserID:  int64(userID),
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

// GetPostHandler handles the fetching of a post along with an image file.
//
// @Summary Retrieve post data along with the associated image
// @Description Fetch post details from Neo4j along with an image file stored in S3. The response contains post details in JSON format followed by the image file.
// @Tags posts
// @Accept json
// @Produce multipart/mixed
// @Param id query string true "Post ID"
// @Header 200 {string} Content-Type "multipart/mixed; boundary=--imageboundary"
// @Success 200 {object} models.Post "Post details with image"
// @Failure 400 {object} errors.ErrResponse "Invalid request or post not found"
// @Failure 500 {object} errors.ErrResponse "Internal server error"
// @Router /api/v1/posts/{id} [get]
func (s *Server) GetPostHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postID := r.URL.Query().Get("id")

	if postID == "" {
		render.Render(w, r, &errors.ErrResponse{
			HTTPStatusCode: http.StatusBadRequest,
			Message:        "postID is empty",
		})
	}

	post, file, err := s.postService.GetPost(ctx, postID)
	if err != nil {
		s.log.ErrorContext(ctx, "Error getting post", slog.Any("error", err))
		render.Render(w, r, errors.ErrInternalServerError)
	}
	defer file.Close()

	// Set headers for file response
	w.Header().Set("Content-Type", "image/jpeg") // Set the appropriate content type
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s", post.ImageName))

	// Write posts data as JSON
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(post)
	if err != nil {
		s.log.ErrorContext(ctx, "Error encoding post", slog.Any("error", err))
		render.Render(w, r, errors.ErrInternalServerError)
		return
	}

	// Stream the image file to the response
	if _, err = io.Copy(w, file); err != nil {
		s.log.ErrorContext(ctx, "Error writing file to response", slog.Any("error", err))
		err = fmt.Errorf("error writing file to response: %w", err)
		render.Render(w, r, errors.ErrInternalServerError)
	}
}
