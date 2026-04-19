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
	"birth_date": "2023-01-15",
	"breed": "Tabby",
	"habits": "Sleeping",
	"weight": 4.5
}`

var TestCatRecord = db.Cat{
	CatID:     1,
	Name:      "Whiskers",
	BirthDate: pgtype.Date{Time: time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)},
	Breed:     "Tabby",
	Weight:    4.5,
	Habits:    "Sleeping",
}

// CreateCat implements Querier
func (m *MockQuerier) CreateCat(
	ctx context.Context,
	params db.CreateCatParams,
) (db.Cat, error) {
	return db.Cat{
		CatID:     1,
		UserID:    params.UserID,
		Name:      params.Name,
		BirthDate: params.BirthDate,
		Breed:     params.Breed,
		Weight:    params.Weight,
		Habits:    params.Habits,
	}, nil
}

// GetCatByID implements Querier
func (m *MockQuerier) GetCatByID(
	ctx context.Context,
	catID int32,
) (db.Cat, error) {
	if catID == 1 {
		return TestCatRecord, nil
	}
	return db.Cat{}, errors.New("cat not found")
}
