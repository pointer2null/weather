// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.24.0

package postgres

import (
	"context"
)

type Querier interface {
	GetAllRecords(ctx context.Context) ([]Weather, error)
	WriteRecord(ctx context.Context, arg WriteRecordParams) error
}

var _ Querier = (*Queries)(nil)
