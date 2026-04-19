package tests

import (
	"time"

	"richmond-api/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
)

// TestCatJSON provides sample cat data in JSON format
const TestCatJSON string = `{
	"name": "Whiskers",
	"birth_date": "2023-01-15",
	"breed": "Tabby",
	"habits": "Sleeping",
	"weight": 4.5
}`

// AddTestData provides a way to add test data to the mock querier
func (m *MockQuerier) AddTestData(
	cats []db.Cat,
	files []db.File,
	sessions map[string]db.Session,
) {
	m.cats = cats
	m.files = files
	m.sessions = sessions
}

// ClearTestData clears all test data from the mock querier
func (m *MockQuerier) ClearTestData() {
	m.cats = nil
	m.files = nil
	m.sessions = make(map[string]db.Session)
}

// TestCatData provides all test data for cats
var TestCatData = struct {
	JSON   string
	Record db.Cat
	Params db.CreateCatParams
}{
	JSON:   TestCatJSON,
	Record: TestCatRecord,
	Params: db.CreateCatParams{
		Name: "Whiskers",
		BirthDate: pgtype.Date{
			Time: time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		Breed:  "Tabby",
		Weight: 4.5,
		Habits: "Sleeping",
	},
}

// TestFileData provides test file data
var TestFileData = struct {
	Content []byte
	Params  db.CreateFileParams
}{
	Content: []byte{
		0xFF,
		0xD8,
		0xFF,
		0xE0,
		0x00,
		0x10,
		0x4A,
		0x46,
		0x49,
		0x46,
		0x00,
	},
	Params: db.CreateFileParams{
		Key:    "test.jpg",
		Size:   11,
		UserID: 42,
		Type:   "image/jpeg",
	},
}

// TestSessionData provides test session data
var TestSessionData = struct {
	Token   string
	UserID  int32
	Session db.Session
}{
	Token:  "test-token",
	UserID: 42,
	Session: db.Session{
		SessionID: 1,
		UserID:    42,
		Token:     "test-token",
	},
}
