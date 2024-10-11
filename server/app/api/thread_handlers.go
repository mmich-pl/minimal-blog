package api

import (
	"encoding/json"
	"errors"
	"github.com/go-chi/render"
	"io"
	"log/slog"
	"ndb/server/app/models"
	apierr "ndb/server/errors"
	"ndb/server/services/posts"
	"net/http"
)

// CreateThreadHandler handles the creation of a new thread
// @Summary Create a new thread
// @Description Create a new thread based on the provided request data
// @Tags threads
// @Accept  json
// @Produce  json
// @Param data body models.CreateThreadRequest true "Thread creation request"
// @Success 200 {object} models.ThreadCreationResponse
// @Failure 400 {object} errors.ErrResponse "Invalid request"
// @Failure 500 {object} errors.ErrResponse "Internal server error"
// @Router /api/v1/threads [post]
func (s *Server) CreateThreadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	b, err := io.ReadAll(r.Body)
	if err != nil {
		s.log.ErrorContext(
			ctx,
			"Error reading body",
			slog.Any("error", err),
		)

		render.Render(w, r, apierr.ErrBadRequest)
		return
	}

	data := &models.CreateThreadRequest{}
	if err = json.Unmarshal(b, data); err != nil {
		s.log.ErrorContext(
			ctx,
			"Failed to parse request while creating thread",
			slog.Any("error", err),
		)
		render.Render(w, r, apierr.ErrBadRequest)
		return
	}

	threadID, err := s.postService.CreateThread(ctx, data)
	if err != nil {
		s.log.ErrorContext(
			ctx,
			"Failed to create thread",
			slog.Any("error", err),
		)
		render.Render(w, r, apierr.ErrInternalServerError)
		return
	}

	render.Respond(w, r, &models.ThreadCreationResponse{
		Status:   http.StatusOK,
		ThreadID: threadID,
	})
}

// ListThreadsHandler fetches the list of threads
// @Summary List all threads
// @Description Fetches a list of all available threads
// @Tags threads
// @Produce  json
// @Success 200 {array} models.Thread "List of threads"
// @Success 404 {object} errors.ErrResponse "Not found error"
// @Failure 500 {object} errors.ErrResponse "Internal server error"
// @Router /api/v1/threads [get]
func (s *Server) ListThreadsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	threads, err := s.postService.ListThreads(ctx)
	if err != nil {
		if errors.Is(err, posts.ErrNotFound) {
			s.log.ErrorContext(
				ctx,
				"No threads found",
				slog.Any("error", err),
			)
			render.Render(w, r, apierr.ErrNotFound)
			return
		}

		s.log.ErrorContext(
			ctx,
			"Failed to fetch list of threads.",
			slog.Any("error", err),
		)

		render.Render(w, r, apierr.ErrInternalServerError)
		return
	}

	render.Respond(w, r, threads)
}

// ListTagsHandler fetches the list of tags
// @Summary List all tags
// @Description Fetches a list of all available tags
// @Tags tags
// @Produce  json
// @Success 200 {array} string "List of tags"
// @Success 404 {object} errors.ErrResponse "Not found error"
// @Failure 500 {object} errors.ErrResponse "Internal server error"
// @Router /api/v1/tags [get]
func (s *Server) ListTagsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	threads, err := s.postService.ListTags(ctx)
	if err != nil {
		if errors.Is(err, posts.ErrNotFound) {
			s.log.ErrorContext(
				ctx,
				"No tags found",
				slog.Any("error", err),
			)
			render.Render(w, r, apierr.ErrNotFound)
			return
		}

		s.log.ErrorContext(
			ctx,
			"Failed to fetch list of threads.",
			slog.Any("error", err),
		)

		render.Render(w, r, apierr.ErrInternalServerError)
		return
	}

	render.Respond(w, r, threads)
}
