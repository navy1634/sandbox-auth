package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq"
	"sandbox-nextjs/api/src/ent"
)

func Connect(ctx context.Context, databaseURL string) (*ent.Client, error) {
	deadline := time.Now().Add(30 * time.Second)

	for {
		db, err := sql.Open("postgres", databaseURL)
		if err != nil {
			if time.Now().After(deadline) {
				return nil, err
			}
			time.Sleep(time.Second)
			continue
		}

		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = db.PingContext(pingCtx)
		cancel()

		if err == nil {
			driver := entsql.OpenDB(dialect.Postgres, db)
			return ent.NewClient(ent.Driver(driver)), nil
		}

		db.Close()
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("database is not ready: %w", err)
		}
		time.Sleep(time.Second)
	}
}
