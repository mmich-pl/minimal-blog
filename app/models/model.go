package models

import "net/http"

type CreatePostRequest struct {
	Title   string `json:"title"`
	UserID  int64  `json:"user_id"`
	Thread  string `json:"thread"`
	Content string `json:"content"`
}

func (mr *CreatePostRequest) Bind(_ *http.Request) error {
	return nil
}

type CreateThreadRequest struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}
