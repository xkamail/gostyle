package pgctx

import "context"

type Row struct{}

func (Row) Scan(dest ...any) error { return nil }

func QueryRow(ctx context.Context, query string, args ...any) *Row { return &Row{} }

func Exec(ctx context.Context, query string, args ...any) (any, error) { return nil, nil }

func Iter(ctx context.Context, f func(scan func(...any) error) error, query string, args ...any) error {
	return nil
}
