package tests

import (
	"context"
	"errors"
	"richmond-api/internal/db"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// TestPost is sample JSON for creating a post
const TestPost string = `{
	"catId": "1",
	"title": "Whiskers playing",
	"body": "Look at this cute cat!"
}`

// TestPostWhiskers is a test post for Whiskers
var TestPostWhiskers = db.Post{
	PostID:    1,
	UserID:    42,
	CatID:     1,
	Title:     "Whiskers playing",
	Body:      "Look at this cute cat!",
	CreatedAt: pgtype.Timestamp{Time: testTime, Valid: true},
	UpdatedAt: pgtype.Timestamp{Valid: false},
}

// testTime is a consistent time for testing
var testTime = time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

// MockPostQuerier implements post.Querier interface for testing
type MockPostQuerier struct {
	posts []db.Post
	files []db.File
	cats  []db.Cat
}

// NewMockPostQuerier creates a new MockPostQuerier
func NewMockPostQuerier() *MockPostQuerier {
	return &MockPostQuerier{
		posts: make([]db.Post, 0),
		files: make([]db.File, 0),
	}
}

// AddPost adds a post for testing
func (m *MockPostQuerier) AddPost(post db.Post) {
	m.posts = append(m.posts, post)
}

// AddFile adds a file for testing
func (m *MockPostQuerier) AddFile(file db.File) {
	m.files = append(m.files, file)
}

// AddCat adds a cat for testing (allows custom UserID)
func (m *MockPostQuerier) AddCat(cat db.Cat) {
	m.cats = append(m.cats, cat)
}

// GetCatByID implements post.Querier
func (m *MockPostQuerier) GetCatByID(
	ctx context.Context,
	catID int32,
) (db.Cat, error) {
	// Look up cat in added cats
	for _, cat := range m.cats {
		if cat.CatID == catID {
			return cat, nil
		}
	}
	// Default: return a test cat that belongs to user 42
	if catID == 1 {
		return db.Cat{
			CatID:  1,
			UserID: 42,
			Name:   "Whiskers",
		}, nil
	}
	return db.Cat{}, errors.New("cat not found")
}

// CreatePost implements post.Querier
func (m *MockPostQuerier) CreatePost(
	ctx context.Context,
	params db.CreatePostParams,
) (db.Post, error) {
	newPost := db.Post{
		PostID:    int32(len(m.posts) + 1),
		UserID:    params.UserID,
		CatID:     params.CatID,
		Title:     params.Title,
		Body:      params.Body,
		CreatedAt: pgtype.Timestamp{Time: testTime, Valid: true},
		UpdatedAt: pgtype.Timestamp{Valid: false},
	}
	m.posts = append(m.posts, newPost)
	return newPost, nil
}

// GetPostByID implements post.Querier
func (m *MockPostQuerier) GetPostByID(
	ctx context.Context,
	postID int32,
) (db.Post, error) {
	for _, post := range m.posts {
		if post.PostID == postID {
			return post, nil
		}
	}
	return db.Post{}, errors.New("post not found")
}

// ListPosts implements post.Querier
func (m *MockPostQuerier) ListPosts(
	ctx context.Context,
	params db.ListPostsParams,
) ([]db.Post, error) {
	if len(m.posts) == 0 {
		return []db.Post{}, nil
	}
	limit := params.Limit
	if limit <= 0 || int(limit) > len(m.posts) {
		limit = int32(len(m.posts))
	}
	offset := params.Offset
	if offset < 0 || int(offset) > len(m.posts) {
		offset = 0
	}
	end := int(offset) + int(limit)
	if end > len(m.posts) {
		end = len(m.posts)
	}
	return m.posts[offset:end], nil
}

// UpdatePost implements post.Querier
func (m *MockPostQuerier) UpdatePost(
	ctx context.Context,
	params db.UpdatePostParams,
) (db.Post, error) {
	for i, post := range m.posts {
		if post.PostID == params.PostID {
			m.posts[i].Title = params.Title
			m.posts[i].Body = params.Body
			return m.posts[i], nil
		}
	}
	return db.Post{}, errors.New("post not found")
}

// DeletePost implements post.Querier
func (m *MockPostQuerier) DeletePost(
	ctx context.Context,
	params db.DeletePostParams,
) (int32, error) {
	for i, post := range m.posts {
		if post.PostID == params.PostID && post.UserID == params.UserID {
			m.posts = append(m.posts[:i], m.posts[i+1:]...)
			return params.PostID, nil
		}
	}
	return 0, errors.New("post not found")
}

// CreateFile implements post.Querier
func (m *MockPostQuerier) CreateFile(
	ctx context.Context,
	params db.CreateFileParams,
) (db.File, error) {
	newFile := db.File{
		ID:        int32(len(m.files) + 1),
		UserID:    params.UserID,
		CatID:     params.CatID,
		PostID:    params.PostID,
		Key:       params.Key,
		Url:       params.Url,
		Width:     params.Width,
		Height:    params.Height,
		Size:      params.Size,
		Quality:   params.Quality,
		Type:      params.Type,
		CreatedAt: pgtype.Timestamp{Time: testTime, Valid: true},
		UpdatedAt: pgtype.Timestamp{Valid: false},
	}
	m.files = append(m.files, newFile)
	return newFile, nil
}

// GetFilesByPostID implements post.Querier
func (m *MockPostQuerier) GetFilesByPostID(
	ctx context.Context,
	postID pgtype.Int4,
) ([]db.File, error) {
	var result []db.File
	for _, file := range m.files {
		if file.PostID.Valid && file.PostID.Int32 == postID.Int32 {
			result = append(result, file)
		}
	}
	if result == nil {
		result = []db.File{}
	}
	return result, nil
}

// MockS3Uploader implements post.S3Uploader interface for testing
type MockS3Uploader struct {
	UploadFunc func(key string, data []byte) (interface{}, error)
}

// Upload implements post.S3Uploader - returns interface{} to satisfy fileutil.Uploader
func (m *MockS3Uploader) Upload(key string, data []byte) (interface{}, error) {
	if m.UploadFunc != nil {
		return m.UploadFunc(key, data)
	}
	// Default no-op behavior
	return "http://test-bucket/" + key, nil
}

// NewMockS3Uploader creates a new MockS3Uploader for testing
func NewMockS3Uploader() *MockS3Uploader {
	return &MockS3Uploader{
		UploadFunc: func(key string, data []byte) (interface{}, error) {
			return "http://test-bucket/" + key, nil
		},
	}
}

// MockS3UploaderWithError creates a mock S3Uploader that returns an error
func MockS3UploaderWithError(err error) *MockS3Uploader {
	return &MockS3Uploader{
		UploadFunc: func(key string, data []byte) (interface{}, error) {
			return nil, err
		},
	}
}
