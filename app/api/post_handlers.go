package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/render"

	"ndb/app/models"
	"ndb/errors"
)

type PostResponse struct {
	Status int    `json:"status"`
	PostID string `json:"post_id"`
}

func (hr PostResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

// CreatePost handles the creation of a new post along with an image upload.
//
// @Summary Create a new post with an image
// @Description This endpoint allows users to create a new post by submitting text data (title, content, thread, user_id) and an image file.
// The image is saved in the user's designated S3 bucket, and the post details are saved in MongoDB.
// @Tags posts
// @Accept multipart/form-data
// @Produce json
// @Param image formData file true "Image File"
// @Param title formData string true "Title of the post"
// @Param content formData string true "Content of the post"
// @Param thread formData string true "ID of the thread to which the post belongs"
// @Param user_id formData integer true "ID of the user creating the post"
// @Success 200 {object} PostResponse
// @Failure 400 {object} errors.ErrResponse "Bad Request"
// @Failure 500 {object} errors.ErrResponse "Internal Server Error"
// @Router /api/v1/posts [post]
func (s *Server) CreatePost(w http.ResponseWriter, r *http.Request) {
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
		render.Render(w, r, &errors.ErrResponse{
			HTTPStatusCode: http.StatusBadRequest,
		})
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
		render.Render(w, r, &errors.ErrResponse{
			Err:            err,
			HTTPStatusCode: http.StatusBadRequest,
			Message:        "Failed to open image file",
		})
		return
	}
	defer file.Close()

	// Process the post creation (similarly as before)
	data := models.CreatePostRequest{
		Title:   title,
		Content: content,
		Thread:  thread,
		UserID:  int64(userID),
	}

	postID, err := s.uploadService.CreatePost(ctx, file, &data)
	if err != nil {
		render.Render(w, r, &errors.ErrResponse{
			Err:            err,
			HTTPStatusCode: http.StatusBadRequest,
			Message:        fmt.Sprintf("error uploading file: %s", err),
		})
		return
	}

	// Return response
	render.Render(w, r, &PostResponse{
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
