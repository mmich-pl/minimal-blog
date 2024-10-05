package posts_test

import (
	"context"
	"log/slog"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/neo4j"

	"ndb/app/models"
	"ndb/config"
	repo "ndb/repositories/posts"
	"ndb/repositories/posts/model"
)

func setupNeo4jContainer(ctx context.Context) (testcontainers.Container, *config.Neo4j, error) {
	// Start a Neo4j container
	neo4jContainer, err := neo4j.Run(ctx,
		"docker.io/neo4j:4.4",
		neo4j.WithAdminPassword("test"),
		neo4j.WithLabsPlugin(neo4j.Apoc),
		neo4j.WithNeo4jSetting("dbms.tx_log.rotation.size", "42M"),
	)
	if err != nil {
		return nil, nil, err
	}

	// Get the mapped port and build the Neo4j driver URL
	host, err := neo4jContainer.Host(ctx)
	if err != nil {
		return nil, nil, err
	}

	port, err := neo4jContainer.MappedPort(ctx, "7687")
	if err != nil {
		return nil, nil, err
	}

	p, err := strconv.Atoi(port.Port())
	if err != nil {
		return nil, nil, err
	}
	return neo4jContainer, &config.Neo4j{
		Host:     host,
		Port:     p,
		Username: "neo4j",
		Password: "test",
	}, nil
}

func TestCreatePost(t *testing.T) {
	ctx := context.Background()

	// Setup Neo4j container
	neo4jContainer, cfg, err := setupNeo4jContainer(ctx)
	require.NoError(t, err)
	defer neo4jContainer.Terminate(ctx)

	// Initialize the Store and Post model
	s, err := repo.NewStore(ctx, slog.Default(), cfg)
	require.NoError(t, err)

	_, err = s.CreateThread(ctx, model.ThreadFrom(&models.CreateThreadRequest{
		Name: "Test Thread",
		Tags: []string{"tag1"},
	}))
	require.NoError(t, err)

	post := model.PostFrom(&models.CreatePostRequest{
		Title:   "Test Post",
		UserID:  123,
		Thread:  "Test Thread",
		Content: "This is a test post content.",
	})

	// Run the CreatePost function
	_, err = s.CreatePost(ctx, post, "Test Thread")
	require.NoError(t, err)

	// Validate that the Post was created in Neo4j
	//result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
	//	query := `
	//		MATCH (p:Post {user_id: $userID, title: $title})-[:BELONGS_TO]->(t:Thread {id: $threadID})
	//		RETURN p.post_id as postID
	//	`
	//	res, err := tx.Run(ctx, query, map[string]interface{}{
	//		"userID":   post.UserID,
	//		"title":    post.Title,
	//		"threadID": "thread1",
	//	})
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	record, err := res.Single(ctx)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	return record.Values[0].(string), nil
	//})
	//require.NoError(t, err)
	//
	//// Assert the postID is the same
	//assert.Equal(t, postID, result)
}
