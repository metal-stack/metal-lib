package auditing

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/lopezator/migrator"

	_ "github.com/lib/pq"
)

type (
	TimescaleDbConfig struct {
		Host     string
		Port     string
		DB       string
		User     string
		Password string

		// Retention defines when audit traces will be thrown away, only settable on initial database usage
		// If this needs to be changed over time, you need to do this manually. Defaults to '14 days'.
		Retention string
		// CompressionInterval defines after which period audit traces will be compressed, only settable on initial database usage.
		// If this needs to be changed over time, you need to do this manually. Defaults to '7 days'.
		CompressionInterval string
		// ChunkInterval defines after which period audit traces will be stored in a new chunk table, only settable on initial database usage.
		// If this needs to be changed over time, you need to do this manually. Defaults to '1 days'.
		ChunkInterval string
	}

	timescaleAuditing struct {
		component string
		db        *sqlx.DB
		log       *slog.Logger

		config *TimescaleDbConfig
	}

	timescaledbRow struct {
		Timestamp time.Time `db:"timestamp"`
		Entry     []byte    `db:"entry"`
	}

	sqlCompOp string
)

const (
	equals sqlCompOp = "equals"
	like   sqlCompOp = "like"
)

func NewTimescaleDB(c Config, tc TimescaleDbConfig) (Auditing, error) {
	if c.Component == "" {
		component, err := defaultComponent()
		if err != nil {
			return nil, err
		}

		c.Component = component
	}

	if tc.Port == "" {
		tc.Port = "5432"
	}
	if tc.DB == "" {
		tc.DB = "postgres"
	}
	if tc.User == "" {
		tc.User = "postgres"
	}
	if tc.Retention == "" {
		tc.Retention = "14 days"
	}
	if tc.ChunkInterval == "" {
		tc.ChunkInterval = "1 days"
	}
	if tc.CompressionInterval == "" {
		tc.CompressionInterval = "7 days"
	}

	source := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable", tc.Host, tc.Port, tc.User, tc.DB, tc.Password)

	db, err := sqlx.Connect("postgres", source)
	if err != nil {
		return nil, fmt.Errorf("could not connect to datastore: %w", err)
	}

	a := &timescaleAuditing{
		component: c.Component,
		log:       c.Log.WithGroup("auditing"),
		db:        db,
		config:    &tc,
	}

	err = a.initialize()
	if err != nil {
		return nil, fmt.Errorf("unable to initialize timescaledb backend: %w", err)
	}

	a.log.Info("connected to timescaledb backend")

	return a, nil
}

func (a *timescaleAuditing) initialize() error {
	type txStatement struct {
		query string
		args  []any
	}

	initialSchema := &migrator.Migration{
		Name: "Initial database schema",
		Func: func(tx *sql.Tx) error {
			for _, stmt := range []txStatement{
				{
					query: `CREATE EXTENSION IF NOT EXISTS timescaledb`,
				},
				{
					query: `CREATE EXTENSION IF NOT EXISTS pg_stat_statements`,
				},
				{
					query: `CREATE TABLE IF NOT EXISTS traces (
						timestamp timestamp NOT NULL,
						entry jsonb NOT NULL
					)`,
				},
				{
					query: `SELECT create_hypertable('traces', 'timestamp', chunk_time_interval => $1::interval, if_not_exists => TRUE)`,
					args:  []any{a.config.ChunkInterval},
				},
				{
					query: `ALTER TABLE traces SET (
						timescaledb.compress,
						timescaledb.compress_orderby = 'timestamp'
					)`,
				},
				{
					query: `SELECT add_compression_policy('traces', $1::interval)`,
					args:  []any{a.config.CompressionInterval},
				},
				{
					query: `CREATE INDEX IF NOT EXISTS traces_gin_idx ON traces USING GIN (entry)`,
				},
				{
					query: `SELECT add_retention_policy('traces', $1::interval)`,
					args:  []any{a.config.Retention},
				},
			} {
				if _, err := tx.Exec(stmt.query, stmt.args...); err != nil {
					return err
				}
			}

			return nil
		},
	}

	a.db.SetMaxIdleConns(5)
	a.db.SetConnMaxLifetime(2 * time.Minute)
	a.db.SetMaxOpenConns(95)

	m, err := migrator.New(
		migrator.WithLogger(migrator.LoggerFunc(func(msg string, args ...interface{}) {
			a.log.Info(fmt.Sprintf(msg, args...))
		})),
		migrator.Migrations(
			initialSchema,
		),
	)
	if err != nil {
		return err
	}

	if err := m.Migrate(a.db.DB); err != nil {
		return err
	}

	return nil
}

func (a *timescaleAuditing) Flush() error {
	return nil
}

func (a *timescaleAuditing) Index(entry Entry) error {
	if entry.Timestamp.IsZero() {
		return errors.New("timestamp is not set")
	}

	q, _, err := sq.
		Insert("traces").
		Columns("timestamp", "entry").
		Values(sq.Expr(":timestamp"), sq.Expr(":entry")).
		ToSql()
	if err != nil {
		return err
	}

	e, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("error marshaling entry: %w", err)
	}

	row := timescaledbRow{
		Timestamp: entry.Timestamp,
		Entry:     e,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = a.db.NamedExecContext(ctx, q, row)
	if err != nil {
		return fmt.Errorf("unable to index audit trace: %w", err)
	}

	return nil
}

func (a *timescaleAuditing) Search(ctx context.Context, filter EntryFilter) ([]Entry, error) {
	var (
		where     []string
		values    = map[string]interface{}{}
		addFilter = func(field string, value any, op sqlCompOp) error {
			if reflect.ValueOf(value).IsZero() {
				return nil
			}

			values[field] = value

			switch op {
			case equals:
				where = append(where, fmt.Sprintf("entry ->> '%s'=:%s", field, field))
			case like:
				where = append(where, fmt.Sprintf("entry ->> '%s' like '%%' || :%s || '%%'", field, field))
			default:
				return fmt.Errorf("comp op not known")
			}

			return nil
		}
	)

	if err := addFilter("body", filter.Body, like); err != nil {
		return nil, err
	}
	if err := addFilter("component", filter.Component, equals); err != nil {
		return nil, err
	}
	if err := addFilter("detail", filter.Detail, equals); err != nil {
		return nil, err
	}
	if err := addFilter("error", filter.Error, equals); err != nil {
		return nil, err
	}
	if err := addFilter("forwardedfor", filter.ForwardedFor, equals); err != nil {
		return nil, err
	}
	if err := addFilter("path", filter.Path, equals); err != nil {
		return nil, err
	}
	if err := addFilter("phase", filter.Phase, equals); err != nil {
		return nil, err
	}
	if err := addFilter("remoteaddr", filter.RemoteAddr, equals); err != nil {
		return nil, err
	}
	if err := addFilter("rqid", filter.RequestId, equals); err != nil {
		return nil, err
	}
	if err := addFilter("statuscode", filter.StatusCode, equals); err != nil {
		return nil, err
	}
	if err := addFilter("tenant", filter.Tenant, equals); err != nil {
		return nil, err
	}
	if err := addFilter("type", filter.Type, equals); err != nil {
		return nil, err
	}
	if err := addFilter("userid", filter.User, equals); err != nil {
		return nil, err
	}

	// to make queries more efficient for timescaledb, we always provide from
	if filter.From.IsZero() {
		filter.From = time.Now().Add(-24 * time.Hour)
	}

	values["from"] = filter.From
	where = append(where, "timestamp >= :from")

	if !filter.To.IsZero() {
		values["to"] = filter.To
		where = append(where, "timestamp <= :to")
	}

	query := sq.
		Select("timestamp", "entry").
		From("traces").
		Where(strings.Join(where, " AND ")).
		OrderBy("timestamp ASC")

	if filter.Limit != 0 {
		query.Limit(uint64(filter.Limit))
	}

	q, _, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := a.db.NamedQueryContext(ctx, q, values)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry

	for rows.Next() {
		var e timescaledbRow

		err = rows.StructScan(&e)
		if err != nil {
			return nil, err
		}

		var entry Entry
		err = json.Unmarshal(e.Entry, &entry)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling entry: %w", err)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}
