package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"go-database-sql-issue/sql"
	"go-database-sql-issue/stdlib"
	"golang.org/x/sync/errgroup"
	"sync/atomic"
	"time"
)

const (
	maxOpenConnections = 200
	maxIdleConnections = 100
	numWorkers         = 300
	timeout            = 3 * time.Second
	queryInterval      = 1 * time.Second
)

func main() {
	cfg, err := pgx.ParseConfig("host=localhost port=5432 user=devbox password=secret dbname=postgres connect_timeout=30")
	if err != nil {
		panic(err)
	}
	db := stdlib.OpenDB(*cfg)
	db.SetMaxIdleConns(maxIdleConnections)
	db.SetMaxOpenConns(maxOpenConnections)
	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)
	var reqCount, errCount atomic.Int64
	g.Go(func() error {
		t := time.NewTicker(1000 * time.Millisecond)
		start := time.Now()

		for {
			select {
			case <-t.C:
				fmt.Printf("time: %vs, err/req (%d/%d), NumConnClosed: %d, NumConnRequests: %d, NumPendingOpenConn: %d, DB stats: %+v\n",
					time.Since(start).Seconds(),
					errCount.Swap(0),
					reqCount.Swap(0),
					db.NumClosed.Load(),
					len(db.ConnRequests),
					len(db.OpenerCh),
					db.Stats())
			case <-ctx.Done():
				t.Stop()
				return ctx.Err()
			}
		}
	})
	for i := 0; i < numWorkers; i++ {
		g.Go(func() error {
			t := time.NewTicker(queryInterval)
			for {
				select {
				case <-ctx.Done():
					t.Stop()
					return ctx.Err()
				case <-t.C:
					ctx, cancel := context.WithTimeout(ctx, timeout)
					defer cancel()
					if err := connectWithDB(ctx, db); err != nil {
						errCount.Add(1)
					}
					reqCount.Add(1)
				}
			}
		})
		time.Sleep(queryInterval / numWorkers)
	}
	g.Wait()
}

func connectWithDB(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var t time.Time
	if err := tx.QueryRowContext(ctx, `select current_timestamp`).Scan(&t); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
