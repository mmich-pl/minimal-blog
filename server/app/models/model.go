package models

import "net/http"

type CreatePostRequest struct {
	Title  string `json:"title"`
	UserID int64  `json:"user_id"`
	Thread string `json:"thread"`
}

func (mr *CreatePostRequest) Bind(_ *http.Request) error {
	return nil
}

type CreateThreadRequest struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

func (hr CreateThreadRequest) Bind(*http.Request) error {
	return nil
}

type PostCreationResponse struct {
	Status int    `json:"status"`
	PostID string `json:"post_id"`
}

func (hr PostCreationResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

type ThreadCreationResponse struct {
	Status   int    `json:"status"`
	ThreadID string `json:"thread_id"`
}

func (mr *ThreadCreationResponse) Bind(_ *http.Request) error {
	return nil
}

type Post struct {
	PostID     string `json:"post_id"`
	UserID     string `json:"user_id"`
	ThreadID   string `json:"thread_id"`
	Title      string `json:"title"`
	ContentFle string `json:"content_fle"`

	ViewCount int `json:"view_count"`
}

func (hr Post) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

type Thread struct {
	ThreadID string   `json:"thread_id"`
	Name     string   `json:"name"`
	Tags     []string `json:"tags"`
}

func (hr Thread) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}
