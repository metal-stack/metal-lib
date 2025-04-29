package auditing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lopezator/migrator"

	_ "github.com/lib/pq"
)

const timescaleDbIndexTimeout = 2 * time.Second

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

		MaxIdleConns    *int
		ConnMaxLifetime *time.Duration
		MaxOpenConns    *int
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
	phrase sqlCompOp = "phrase"
)

func NewTimescaleDB(c Config, tc TimescaleDbConfig) (Auditing, error) {
	if c.Component == "" {
		component, err := defaultComponent()
		if err != nil {
			return nil, err
		}

		c.Component = component
	}

	if c.Async {
		return nil, fmt.Errorf("timescaledb backend does not support async indexing")
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
		log:       c.Log.WithGroup("auditing").With("audit-backend", "timescaledb"),
		db:        db,
		config:    &tc,
	}

	maxIdleConns := 5
	if tc.MaxIdleConns != nil {
		maxIdleConns = *tc.MaxIdleConns
	}
	a.db.SetMaxIdleConns(maxIdleConns)

	connMaxLifetime := 2 * time.Minute
	if tc.ConnMaxLifetime != nil {
		connMaxLifetime = *tc.ConnMaxLifetime
	}
	a.db.SetConnMaxLifetime(connMaxLifetime)

	maxOpenConns := 95
	if tc.MaxOpenConns != nil {
		maxOpenConns = *tc.MaxOpenConns
	}
	a.db.SetMaxOpenConns(maxOpenConns)

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
						entry jsonb NOT NULL,
						ts tsvector GENERATED ALWAYS AS (to_tsvector('simple', entry)) STORED
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
					query: `CREATE INDEX IF NOT EXISTS ts_idx ON traces USING GIN (ts)`,
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
	if entry.Component == "" {
		entry.Component = a.component
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	q := "INSERT INTO traces (timestamp, entry) VALUES (:timestamp, :entry)"

	e, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("error marshaling entry: %w", err)
	}

	row := timescaledbRow{
		Timestamp: entry.Timestamp.UTC(),
		Entry:     e,
	}

	ctx, cancel := context.WithTimeout(context.Background(), timescaleDbIndexTimeout)
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
			case phrase:
				// the additional "like" match allows matching partial words, too
				where = append(where, fmt.Sprintf("(ts @@ websearch_to_tsquery('simple', '$$' || :%s || '$$') or entry ->> '%s' like '%%' || :%s || '%%')", field, field, field))
			default:
				return fmt.Errorf("comp op not known")
			}

			return nil
		}
	)

	if err := addFilter("body", filter.Body, phrase); err != nil {
		return nil, err
	}
	if err := addFilter("component", filter.Component, equals); err != nil {
		return nil, err
	}
	if err := addFilter("detail", filter.Detail, equals); err != nil {
		return nil, err
	}
	if err := addFilter("error", filter.Error, phrase); err != nil {
		return nil, err
	}
	if err := addFilter("forwardedfor", filter.ForwardedFor, like); err != nil {
		return nil, err
	}
	if err := addFilter("path", filter.Path, like); err != nil {
		return nil, err
	}
	if err := addFilter("phase", filter.Phase, equals); err != nil {
		return nil, err
	}
	if err := addFilter("remoteaddr", filter.RemoteAddr, like); err != nil {
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
	if err := addFilter("project", filter.Project, equals); err != nil {
		return nil, err
	}
	if err := addFilter("type", filter.Type, equals); err != nil {
		return nil, err
	}
	if err := addFilter("user", filter.User, equals); err != nil {
		return nil, err
	}

	// to make queries more efficient for timescaledb, we always provide from
	if filter.From.IsZero() {
		filter.From = time.Now().Add(-24 * time.Hour).UTC()
	}

	values["from"] = filter.From.UTC()
	where = append(where, "timestamp >= :from")

	if !filter.To.IsZero() {
		values["to"] = filter.To.UTC()
		where = append(where, "timestamp <= :to")
	}

	q := "SELECT timestamp,entry FROM traces"
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY timestamp ASC"

	if filter.Limit != 0 {
		q += fmt.Sprintf(" LIMIT %d", filter.Limit)
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
			return nil, fmt.Errorf("error unmarshalling entry: %w", err)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}
