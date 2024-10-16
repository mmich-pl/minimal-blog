package posts

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"log/slog"
	"ndb/server/config"
	"ndb/server/repositories/posts/model"
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

	thread.ThreadID = uuid.New().String()
	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(
			ctx,
			`CREATE (t:Thread {
				threadID: $id,				
				name: $name,
				createdAt: $createdAt,
                updatedAt: $updatedAt
				}) RETURN t`,
			map[string]any{
				"id":        thread.ThreadID,
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

		_, err = res.Single(ctx)
		if err != nil {
			return nil, err
		}

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

		return thread.ThreadID, nil
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

	post.PostID = uuid.New().String()
	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Neo4j query to create the Post and connect it to the Thread node
		query := `MATCH (t:Thread {threadID: $thread})
            CREATE (p:Post {
				postID: $id,
                userID: $userID,
                title: $title,
                contentFile: $contentFile,
                viewCount: $viewCount,
                status: $status,
                createdAt: $createdAt,
                updatedAt: $updatedAt
            })-[:BELONGS_TO]->(t)
            RETURN p`

		// Run the query with all posts data
		_, err := tx.Run(
			ctx,
			query,
			map[string]any{
				"id":          post.PostID,
				"userID":      post.UserID,
				"title":       post.Title,
				"contentFile": post.ContentFile,
				"viewCount":   post.ViewCount,
				"status":      post.Status,
				"createdAt":   post.CreatedAt,
				"updatedAt":   post.UpdatedAt,
				"thread":      threadID,
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

		s.log.InfoContext(
			ctx,
			"New posts created successfully",
			slog.Any("posts", post.Title),
		)

		return post.PostID, nil
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

		node := record.Values[0].(neo4j.Node)
		return mapToPost(&node), nil
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
            MATCH (p:Post)-[:BELONGS_TO]->(t:Thread {threadID: $threadID})
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
			node := res.Record().Values[0].(neo4j.Node)
			posts = append(posts, mapToPost(&node))
		}

		return posts, nil
	})

	if err != nil {
		return nil, err
	}
	return result.([]*model.Post), nil
}

func (s *Store) GetPostsWithLimit(ctx context.Context, limit int) (map[string][]*model.Post, error) {
	session := s.conn.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
           MATCH (t:Thread)
OPTIONAL MATCH (t)-[:HAS_TAG]->(tag:Tag)
WITH t, collect(tag.name) AS tags
OPTIONAL MATCH (p:Post)-[:BELONGS_TO]->(t)
  WHERE p.status = 'published'
RETURN t.name AS thread_name, t.threadID, tags, collect(p)[..$limit] AS posts`

		res, err := tx.Run(ctx, query, map[string]interface{}{
			"limit": limit,
		})
		if err != nil {
			return nil, err
		}

		var posts = make(map[string][]*model.Post) // Initialize the map
		for res.Next(ctx) {
			record := res.Record()

			// Get the key for the posts map (from the first value in the record)
			key := record.Values[0].(string)

			// Iterate over the posts in the 6th column (index 5)
			for _, post := range record.Values[3].([]interface{}) {
				p := post.(neo4j.Node)
				mapped := mapToPost(&p)
				mapped.ThreadID = record.Values[1].(string)

				// Check if the key already exists in the map
				if _, exists := posts[key]; !exists {
					// If not, initialize an empty array for this key
					posts[key] = []*model.Post{}
				}

				// Append the mapped post to the array
				posts[key] = append(posts[key], mapped)
			}
		}

		return posts, nil
	})

	if err != nil {
		return nil, err
	}
	return result.(map[string][]*model.Post), nil
}

func mapToPost(node *neo4j.Node) *model.Post {
	post := model.Post{
		PostID:      node.Props["postID"].(string),
		UserID:      node.Props["userID"].(string),
		Title:       node.Props["title"].(string),
		ContentFile: node.Props["contentFile"].(string),
		ViewCount:   int(node.Props["viewCount"].(int64)),
		Status:      model.PostStatus(node.Props["status"].(string)),
		CreatedAt:   node.Props["createdAt"].(string),
		UpdatedAt:   node.Props["updatedAt"].(string),
	}
	return &post
}
