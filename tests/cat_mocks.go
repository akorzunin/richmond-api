package tests

import (
	"context"
	"errors"
	"richmond-api/internal/db"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const TestCat string = `{
	"name": "Whiskers",
	"birthDate": "2023-01-15",
	"breed": "Tabby",
	"habits": "Sleeping",
	"weight": 4.5
}`

var TestCatWhiskers = db.Cat{
	UserID:    1,
	CatID:     1,
	Name:      "Whiskers",
	BirthDate: pgtype.Date{Time: time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)},
	Breed:     "Tabby",
	Weight:    4.5,
	Habits:    "Sleeping",
}

var TestCatMittens = db.Cat{
	CatID:  2,
	UserID: 1,
	Name:   "Mittens",
	Breed:  "Siamese",
	BirthDate: pgtype.Date{
		Time:  time.Date(2022, 6, 20, 0, 0, 0, 0, time.UTC),
		Valid: true,
	},
	Weight: 3.8,
	Habits: "Playing",
}

// CreateCat implements Querier
func (m *MockQuerier) CreateCat(
	ctx context.Context,
	params db.CreateCatParams,
) (db.Cat, error) {
	return db.Cat{}, errors.New("not implemented")
}

// GetCatByID implements Querier
func (m *MockQuerier) GetCatByID(
	ctx context.Context,
	catID int32,
) (db.Cat, error) {
	for _, cat := range m.cats {
		if cat.CatID == catID {
			return cat, nil
		}
	}
	return db.Cat{}, errors.New("cat not found")
}
