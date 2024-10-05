package posts

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

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

	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// Create the thread
		res, err := tx.Run(
			ctx,
			`CREATE (t:Thread {
				name: $name, 
				createdAt: $createdAt,
                updatedAt: $updatedAt
				}) RETURN t`,
			map[string]interface{}{
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
				map[string]interface{}{
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

func (s *Store) CreatePost(ctx context.Context, post *model.Post, threadName string) (string, error) {
	session := s.conn.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// Neo4j query to create the Post and connect it to the Thread node
		query := `MATCH (t:Thread {name: $thread})
            CREATE (p:Post {
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

		// Run the query with all post data
		res, err := tx.Run(
			ctx,
			query,
			map[string]interface{}{
				"userID":    post.UserID,
				"title":     post.Title,
				"content":   post.Content,
				"imageName": post.ImageName,
				"viewCount": post.ViewCount,
				"status":    post.Status,
				"createdAt": post.CreatedAt,
				"updatedAt": post.UpdatedAt,
				"thread":    threadName,
			},
		)
		if err != nil {
			s.log.ErrorContext(
				ctx,
				"Failed to create post",
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
			"New post created successfully",
			slog.Any("post", post.Title),
		)

		// Return the PostID
		return record.Values[0].(neo4j.Node).Props["title"].(string), nil
	})
	if err != nil {
		return "", err
	}
	return result.(string), nil
}
