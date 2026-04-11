// Package dbutil provides shared conversion helpers.
package dbutil

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// PgUUIDToPtr converts a pgtype.UUID to *uuid.UUID
func PgUUIDToPtr(p pgtype.UUID) *uuid.UUID {
	if !p.Valid {
		return nil
	}
	u := uuid.UUID(p.Bytes)
	return &u
}

// UUIDToPgtype converts *uuid.UUID to pgtype.UUID
func UUIDToPgtype(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *u, Valid: true}
}

// PgTimestamptzToPtr converts pgtype.Timestamptz to *time.Time
func PgTimestamptzToPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// PgInt4ToPtr converts pgtype.Int4 to *int
func PgInt4ToPtr(p pgtype.Int4) *int {
	if !p.Valid {
		return nil
	}
	v := int(p.Int32)
	return &v
}

// IntToPgInt4 converts *int to pgtype.Int4
func IntToPgInt4(i *int) pgtype.Int4 {
	if i == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*i), Valid: true}
}
