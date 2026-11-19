package sqlargs

import (
	"context"

	"github.com/acoshift/pgsql/pgctx"
)

func ok(ctx context.Context, id string) {
	var a, b string
	_ = pgctx.QueryRow(ctx, `
		select a, b
		from t
		where id = $1
	`,
		id,
	).Scan(
		&a,
		&b,
	)
	_ = pgctx.QueryRow(ctx, `select a from t where id = $1`, id).Scan(&a)
	_, _ = pgctx.Exec(ctx, `
		delete from t
	`)
	_ = pgctx.QueryRow(ctx, `
		select a
		from t
	`).Scan(
		&a,
	)
	_ = pgctx.Iter(ctx, func(scan func(...any) error) error { return nil }, `
		select a
		from t
		where id = $1
	`,
		id,
	)
}

func bad(ctx context.Context, id string) {
	var a, b string
	_ = pgctx.QueryRow(ctx, `
		select a, b
		from t
		where id = $1
	`, id).Scan( // want `put each query arg on its own line`
		&a, &b, // want `put each Scan arg on its own line`
	)
	_, _ = pgctx.Exec(ctx, `
		update t set a = $2
		where id = $1
	`,
		id, a, // want `put each query arg on its own line`
	)
	_, _ = pgctx.Exec(ctx, `
		update t set a = $2
		where id = $1
	`,
		id,
		a) // want `put \) after the last query arg on its own line`
	_ = pgctx.QueryRow(ctx, `select a, b from t`).Scan(&a, &b) // want `put each Scan arg on its own line`
	_ = pgctx.QueryRow(ctx, `
		select a
		from t
		where id = $1
	`,
		id,
	).Scan(&a) // want `put each Scan arg on its own line`
}
