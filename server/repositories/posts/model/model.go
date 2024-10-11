package model

import (
	"strconv"
	"time"

	"ndb/server/app/models"
)

type PostStatus string

const (
	StatusPublished PostStatus = "published"
	StatusPrivate   PostStatus = "private"
	StatusDeleted   PostStatus = "deleted"
)

type Post struct {
	PostID   string
	UserID   string
	ThreadID string
	Title    string

	ContentFile string

	ViewCount int
	Status    PostStatus
	CreatedAt string
	UpdatedAt string
	DeletedAt string
}

func PostFrom(post *models.CreatePostRequest) *Post {
	return &Post{
		UserID:   strconv.FormatInt(post.UserID, 10),
		ThreadID: post.Thread,
		Title:    post.Title,

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
	ThreadID string
	Name     string
	Tags     []string

	CreatedAt string
	UpdatedAt string
	DeletedAt string
}

func ThreadFrom(thread *models.CreateThreadRequest) *Thread {
	return &Thread{
		Name: thread.Name,
		Tags: thread.Tags,

		CreatedAt: getValidTime().Format(time.RFC3339),
		UpdatedAt: getValidTime().Format(time.RFC3339),
	}
}
