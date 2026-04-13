package s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/minio/minio-go/v7"
)

// mockGetObject simulates an S3 GetObject response.
type mockGetObject struct {
	data []byte
	err  error
}

func (m *mockGetObject) Read(p []byte) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	if len(m.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, m.data)
	m.data = m.data[n:]
	if len(m.data) == 0 {
		return n, io.EOF
	}
	return n, nil
}

func (m *mockGetObject) Close() error {
	return nil
}

// mockClient implements enough of minio.Client for tests.
type mockClient struct {
	putObjectErr error
	getObjectErr error
	getObjectRet *mockGetObject
}

func (m *mockClient) PutObject(ctx context.Context, bucket, key string, data io.Reader, size int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	if m.putObjectErr != nil {
		return minio.UploadInfo{}, m.putObjectErr
	}
	return minio.UploadInfo{}, nil
}

func (m *mockClient) GetObject(ctx context.Context, bucket, key string, opts minio.GetObjectOptions) (io.ReadCloser, error) {
	if m.getObjectErr != nil {
		return nil, m.getObjectErr
	}
	return m.getObjectRet, nil
}

func TestCreateFile(t *testing.T) {
	tests := []struct {
		name    string
		client  *minio.Client
		bucket  string
		key     string
		data    []byte
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil client",
			client:  nil,
			bucket:  "test-bucket",
			key:     "test-key",
			data:    []byte("test data"),
			wantErr: true,
			errMsg:  "S3 client is nil",
		},
		{
			name:    "empty bucket",
			client:  &minio.Client{},
			bucket:  "",
			key:     "test-key",
			data:    []byte("test data"),
			wantErr: true,
			errMsg:  "bucket name is required",
		},
		{
			name:    "empty key",
			client:  &minio.Client{},
			bucket:  "test-bucket",
			key:     "",
			data:    []byte("test data"),
			wantErr: true,
			errMsg:  "object key is required",
		},
		{
			name:    "empty data",
			client:  &minio.Client{},
			bucket:  "test-bucket",
			key:     "test-key",
			data:    nil,
			wantErr: true,
			errMsg:  "data is empty",
		},
		{
			name:    "empty data slice",
			client:  &minio.Client{},
			bucket:  "test-bucket",
			key:     "test-key",
			data:    []byte{},
			wantErr: true,
			errMsg:  "data is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't easily test the success path without a real minio.Client
			// but we can verify validation errors
			if tt.wantErr {
				err := CreateFile(tt.client, tt.bucket, tt.key, tt.data)
				if err == nil {
					t.Errorf("CreateFile() expected error containing %q, got nil", tt.errMsg)
				} else if !bytes.Contains([]byte(err.Error()), []byte(tt.errMsg)) {
					t.Errorf("CreateFile() error %q does not contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestGetFile(t *testing.T) {
	tests := []struct {
		name    string
		client  *minio.Client
		bucket  string
		key     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil client",
			client:  nil,
			bucket:  "test-bucket",
			key:     "test-key",
			wantErr: true,
			errMsg:  "S3 client is nil",
		},
		{
			name:    "empty bucket",
			client:  &minio.Client{},
			bucket:  "",
			key:     "test-key",
			wantErr: true,
			errMsg:  "bucket name is required",
		},
		{
			name:    "empty key",
			client:  &minio.Client{},
			bucket:  "test-bucket",
			key:     "",
			wantErr: true,
			errMsg:  "object key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetFile(tt.client, tt.bucket, tt.key)
			if err == nil {
				t.Errorf("GetFile() expected error containing %q, got nil", tt.errMsg)
			} else if !bytes.Contains([]byte(err.Error()), []byte(tt.errMsg)) {
				t.Errorf("GetFile() error %q does not contain %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestNewReadSeeker(t *testing.T) {
	data := []byte("hello world")
	rs := NewReadSeeker(data)

	// Test Read
	buf := make([]byte, 5)
	n, err := rs.Read(buf)
	if err != nil {
		t.Errorf("Read() error = %v", err)
	}
	if n != 5 {
		t.Errorf("Read() n = %d, want 5", n)
	}
	if string(buf) != "hello" {
		t.Errorf("Read() buf = %q, want %q", string(buf), "hello")
	}

	// Test Seek
	pos, err := rs.Seek(0, io.SeekStart)
	if err != nil {
		t.Errorf("Seek() error = %v", err)
	}
	if pos != 0 {
		t.Errorf("Seek() pos = %d, want 0", pos)
	}

	// Test Read after Seek
	n, err = rs.Read(buf)
	if err != nil {
		t.Errorf("Read() after Seek error = %v", err)
	}
	if n != 5 {
		t.Errorf("Read() after Seek n = %d, want 5", n)
	}
	if string(buf) != "hello" {
		t.Errorf("Read() after Seek buf = %q, want %q", string(buf), "hello")
	}

	// Test SeekEnd
	pos, err = rs.Seek(0, io.SeekEnd)
	if err != nil {
		t.Errorf("Seek(End) error = %v", err)
	}
	if pos != int64(len(data)) {
		t.Errorf("Seek(End) pos = %d, want %d", pos, len(data))
	}

	// Test Read at EOF
	_, err = rs.Read(buf)
	if !errors.Is(err, io.EOF) {
		t.Errorf("Read() at EOF error = %v, want io.EOF", err)
	}
}
