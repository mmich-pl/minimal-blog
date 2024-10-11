package posts

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"log/slog"
	"ndb/config"
	"ndb/repositories/posts/model"
)

type Store struct {
	conn neo4j.DriverWithContext
	log  *slog.Logger
}

func NewStore(
	ctx context.Context,
	logger *slog.Logger,
	cfg *config.Neo4j,
) (*Store, error) {
	uri := fmt.Sprintf("neo4j://%s:%d", cfg.Host, cfg.Port)
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(cfg.Username, cfg.Password, ""))
	if err != nil {
		return nil, err
	}

	logger.DebugContext(
		ctx,
		"Successfully created neo4j driver",
		slog.Any("host", cfg.Host),
		slog.Any("port", cfg.Port),
	)

	return &Store{conn: driver, log: logger}, nil
}

func (s *Store) CreateThread(ctx context.Context, thread *model.Thread) (string, error) {
	session := s.conn.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Create the thread
		res, err := tx.Run(
			ctx,
			`CREATE (t:Thread {
				threadID: $id,				
				name: $name,
				createdAt: $createdAt,
                updatedAt: $updatedAt
				}) RETURN t`,
			map[string]any{
				"id":        uuid.New().String(),
				"name":      thread.Name,
				"createdAt": thread.CreatedAt,
				"updatedAt": thread.UpdatedAt,
			},
		)
		if err != nil {
			s.log.ErrorContext(
				ctx,
				"Failed to create thread",
				slog.Any("error", err),
				slog.Any("thread", thread.Name),
			)
			return nil, err
		}

		s.log.InfoContext(
			ctx,
			"Thread created successfully",
			slog.Any("thread", thread.Name),
		)

		record, err := res.Single(ctx)
		if err != nil {
			return nil, err
		}

		threadID := record.Values[0].(neo4j.Node).ElementId // Retrieve thread ID

		// For each tag, either create a new tag or connect to an existing one
		for _, tag := range thread.Tags {
			_, err = tx.Run(ctx,
				`MERGE (tag:Tag {name: $tagName})
                 WITH tag
                 MATCH (t:Thread {name: $threadName})
                 MERGE (t)-[:HAS_TAG]->(tag)`,
				map[string]any{
					"tagName":    tag,
					"threadName": thread.Name,
				})
			if err != nil {
				s.log.ErrorContext(
					ctx,
					"Failed to add tag",
					slog.Any("error", err),
					slog.Any("thread", thread.Name),
					slog.Any("tag", tag),
				)
				return nil, err
			}
			s.log.InfoContext(
				ctx,
				"New tag added successfully",
				slog.Any("thread", thread.Name),
				slog.Any("tag", tag),
			)
		}

		return threadID, nil
	})
	if err != nil {
		return "", err
	}
	return result.(string), nil
}

func (s *Store) ListThreads(ctx context.Context) ([]*model.Thread, error) {
	session := s.conn.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(
			ctx,
			`MATCH (t: Thread)
					OPTIONAL MATCH (t)-[:HAS_TAG]->(tag:Tag)
					RETURN t.name AS name,
					t.threadID as id,
					t.createdAt as created_at,
					t.updatedAt as updated_at,
					collect(tag.name) AS tags;`,
			nil,
		)

		if err != nil {
			s.log.ErrorContext(
				ctx,
				"Failed to fetch threads",
				slog.Any("error", err),
			)
			return nil, err
		}

		var threads []*model.Thread
		for res.Next(ctx) {
			record := res.Record()
			s.log.InfoContext(ctx, fmt.Sprintf("Found thread: %+v", record))

			rawTags := record.Values[4].([]interface{})
			tags := make([]string, len(rawTags))
			for i, v := range rawTags {
				tags[i] = fmt.Sprint(v)
			}

			threads = append(
				threads,
				&model.Thread{
					Name:      record.Values[0].(string),
					ThreadID:  record.Values[1].(string),
					CreatedAt: record.Values[2].(string),
					UpdatedAt: record.Values[3].(string),
					Tags:      tags,
				},
			)

		}

		return threads, nil
	})
	if err != nil {
		return nil, err
	}
	return result.([]*model.Thread), nil
}

func (s *Store) ListTags(ctx context.Context) ([]string, error) {
	session := s.conn.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	query := "MATCH (t:Tag) RETURN t.name AS tag_name"
	result, err := session.Run(ctx, query, nil)
	if err != nil {
		s.log.ErrorContext(
			ctx,
			"Failed to fetch threads",
			slog.Any("error", err),
		)
		return nil, err
	}

	var tags []string
	for result.Next(ctx) {
		record := result.Record()
		if tagName, ok := record.Get("tag_name"); ok {
			tags = append(tags, tagName.(string))
		}
	}

	return tags, nil
}

func (s *Store) CreatePost(ctx context.Context, post *model.Post, threadID string) (string, error) {
	session := s.conn.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Neo4j query to create the Post and connect it to the Thread node
		query := `MATCH (t:Thread {threadID: $thread})
            CREATE (p:Post {
				postID: $id,
                userID: $userID,
                title: $title,
                content: $content,
                imageName: $imageName,
                viewCount: $viewCount,
                status: $status,
                createdAt: $createdAt,
                updatedAt: $updatedAt
            })-[:BELONGS_TO]->(t)
            RETURN p`

		// Run the query with all posts data
		res, err := tx.Run(
			ctx,
			query,
			map[string]any{
				"id":        uuid.New().String(),
				"userID":    post.UserID,
				"title":     post.Title,
				"content":   post.Content,
				"imageName": post.ImageName,
				"viewCount": post.ViewCount,
				"status":    post.Status,
				"createdAt": post.CreatedAt,
				"updatedAt": post.UpdatedAt,
				"thread":    threadID,
			},
		)
		if err != nil {
			s.log.ErrorContext(
				ctx,
				"Failed to create posts",
				slog.Any("error", err),
			)
			return nil, err
		}

		record, err := res.Single(ctx)
		if err != nil {
			return nil, err
		}

		s.log.InfoContext(
			ctx,
			"New posts created successfully",
			slog.Any("posts", post.Title),
		)

		// Return the ThreadID
		return record.Values[0].(neo4j.Node).Props["postID"].(string), nil
	})
	if err != nil {
		return "", err
	}
	return result.(string), nil
}

func (s *Store) GetPost(ctx context.Context, postID string) (*model.Post, error) {
	session := s.conn.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
            MATCH (p:Post {postID: $postID})
            WHERE p.status = 'published'
            RETURN p`

		res, err := tx.Run(ctx, query, map[string]interface{}{
			"postID": postID,
		})
		if err != nil {
			return nil, err
		}

		record, err := res.Single(ctx)
		if err != nil {
			return nil, err
		}

		return mapToPost(record), nil
	})

	if err != nil {
		return nil, err
	}
	return result.(*model.Post), nil
}

func (s *Store) GetPostsInThread(ctx context.Context, threadID string) ([]*model.Post, error) {
	session := s.conn.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
            MATCH (p:Post)-[:BELONGS_TO]->(t:Thread {id: $threadID})
            WHERE p.status = 'published'
            RETURN p`

		res, err := tx.Run(ctx, query, map[string]interface{}{
			"threadID": threadID,
		})
		if err != nil {
			return nil, err
		}

		var posts []*model.Post
		for res.Next(ctx) {
			record := res.Record()
			posts = append(posts, mapToPost(record))
		}

		return posts, nil
	})

	if err != nil {
		return nil, err
	}
	return result.([]*model.Post), nil
}

func mapToPost(record *neo4j.Record) *model.Post {
	fmt.Printf("%+v\n", record)
	node := record.Values[0].(neo4j.Node)
	post := model.Post{
		PostID:    node.Props["postID"].(string),
		UserID:    node.Props["userID"].(string),
		Title:     node.Props["title"].(string),
		Content:   node.Props["content"].(string),
		ImageName: node.Props["imageName"].(string),
		ViewCount: int(node.Props["viewCount"].(int64)),
		Status:    model.PostStatus(node.Props["status"].(string)),
		CreatedAt: node.Props["createdAt"].(string),
		UpdatedAt: node.Props["updatedAt"].(string),
	}
	return &post
}
