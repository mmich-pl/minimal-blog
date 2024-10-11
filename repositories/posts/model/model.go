package model

import (
	"strconv"
	"time"

	"ndb/app/models"
)

type PostStatus string

const (
	StatusPublished PostStatus = "published"
	StatusPrivate   PostStatus = "private"
	StatusDeleted   PostStatus = "deleted"
)

type Post struct {
	PostID   string `json:"post_id"`
	UserID   string `json:"user_id"`
	ThreadID string `json:"thread_id"`
	Title    string `json:"title"`
	Content  string `json:"content"`

	ImageName string `json:"image_name"`

	ViewCount int        `json:"view_count"`
	Status    PostStatus `json:"status"`
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at"`
	DeletedAt string     `json:"deleted_at"`
}

func PostFrom(post *models.CreatePostRequest) *Post {
	return &Post{
		UserID:   strconv.FormatInt(post.UserID, 10),
		ThreadID: post.Thread,
		Title:    post.Title,
		Content:  post.Content,

		ViewCount: 0,
		Status:    StatusPublished,
		CreatedAt: getValidTime().Format(time.RFC3339),
		UpdatedAt: getValidTime().Format(time.RFC3339),
	}
}

func getValidTime() time.Time {
	loc, _ := time.LoadLocation("UTC") // Use a valid timezone like "UTC"
	return time.Now().In(loc)
}

type Thread struct {
	ThreadID string   `json:"id"`
	Name     string   `json:"name"`
	Tags     []string `json:"tags"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	DeletedAt string `json:"deleted_at"`
}

func ThreadFrom(thread *models.CreateThreadRequest) *Thread {
	return &Thread{
		Name: thread.Name,
		Tags: thread.Tags,

		CreatedAt: getValidTime().Format(time.RFC3339),
		UpdatedAt: getValidTime().Format(time.RFC3339),
	}
}
