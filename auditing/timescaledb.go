package auditing

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
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
	}

	timescaleAuditing struct {
		component string
		db        *sqlx.DB
		log       *slog.Logger

		cols []string
		vals []any
	}

	// to keep the public interface free from field tags like "db" and "json" (as these might differ for different dbs)
	// we introduce an internal type. unfortunately, this requires a conversion, which takes effort to maintain
	timescaleEntry struct {
		Component    string      `db:"component"`
		RequestId    string      `db:"rqid" json:"rqid"`
		Type         EntryType   `db:"type"`
		Timestamp    time.Time   `db:"timestamp"`
		User         string      `db:"userid"`
		Tenant       string      `db:"tenant"`
		Detail       EntryDetail `db:"detail"`
		Phase        EntryPhase  `db:"phase"`
		Path         string      `db:"path"`
		ForwardedFor string      `db:"forwardedfor"`
		RemoteAddr   string      `db:"remoteaddr"`
		Body         any         `db:"body"`
		StatusCode   int         `db:"statuscode"`
		Error        string      `db:"error" json:"-"`
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

	source := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable", tc.Host, tc.Port, tc.User, tc.DB, tc.Password)

	db, err := sqlx.Connect("postgres", source)
	if err != nil {
		return nil, fmt.Errorf("could not connect to datastore: %w", err)
	}

	a := &timescaleAuditing{
		component: c.Component,
		log:       c.Log.WithGroup("auditing"),
		db:        db,
	}

	err = a.initialize()
	if err != nil {
		return nil, fmt.Errorf("unable to initialize timescaledb backend: %w", err)
	}

	a.log.Info("connected to timescaledb backend")

	return a, nil
}

func (a *timescaleAuditing) initialize() error {
	initialSchema := &migrator.Migration{
		Name: "Initial database schema",
		Func: func(tx *sql.Tx) error {
			schema := `
			CREATE EXTENSION IF NOT EXISTS timescaledb;
			CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

			CREATE TABLE IF NOT EXISTS traces (
				timestamp timestamp NOT NULL,
				rqid text NOT NULL,
				component text NOT NULL,
				type text NOT NULL,
				body text NOT NULL,
				error text NOT NULL,
				statuscode int NOT NULL,
				remoteaddr text NOT NULL,
				forwardedfor text NOT NULL,
				path text NOT NULL,
				phase text NOT NULL,
				detail text NOT NULL,
				tenant text NOT NULL,
				userid text NOT NULL
			);

			SELECT create_hypertable('traces', 'timestamp', chunk_time_interval => INTERVAL '1 days', if_not_exists => TRUE);
			ALTER TABLE traces SET (
				timescaledb.compress,
				timescaledb.compress_segmentby = 'rqid',
				timescaledb.compress_orderby = 'timestamp',
				timescaledb.compress_chunk_time_interval = '7 days'
			);
			`
			// TODO: evaluate what is needed
			// CREATE INDEX IF NOT EXISTS traces_idx ON traces();

			if _, err := tx.Exec(schema); err != nil {
				return err
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

	q, _, err := sq.
		Select("column_name").
		From("information_schema.columns").
		Where("table_name='traces'").
		ToSql()
	if err != nil {
		return err
	}

	rows, err := a.db.Query(q)
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Err() != nil {
		return rows.Err()
	}

	for rows.Next() {
		var col string

		err = rows.Scan(&col)
		if err != nil {
			return err
		}

		a.cols = append(a.cols, col)
		a.vals = append(a.vals, sq.Expr(":"+col))
	}

	return nil
}

func (a *timescaleAuditing) Flush() error {
	return nil
}

func (a *timescaleAuditing) Index(entry Entry) error {
	q, _, err := sq.
		Insert("traces").
		Columns(a.cols...).
		Values(a.vals...).
		ToSql()
	if err != nil {
		return err
	}

	internalEntry, err := a.toInternal(entry)
	if err != nil {
		return fmt.Errorf("unable to convert audit trace to database entry: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = a.db.NamedExecContext(ctx, q, internalEntry)
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

			if !slices.Contains(a.cols, field) {
				return fmt.Errorf("unable to filter for %q, no such table column", field)
			}

			values[field] = value

			switch op {
			case equals:
				where = append(where, fmt.Sprintf("%s=:%s", field, field))
			case like:
				where = append(where, fmt.Sprintf("%s like '%%' || %s || '%%'", field, field))
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

	query := sq.
		Select(a.cols...).
		From("traces").
		Columns(a.cols...).
		Where(strings.Join(where, " AND ")).
		OrderBy("timestamp ASC")

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
		var e timescaleEntry

		err = rows.StructScan(&e)
		if err != nil {
			return nil, err
		}

		entry, err := a.toExternal(e)
		if err != nil {
			return nil, fmt.Errorf("unable to convert entry: %w", err)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func (_ *timescaleAuditing) toInternal(e Entry) (*timescaleEntry, error) {
	intermediate, err := json.Marshal(e) // nolint
	if err != nil {
		return nil, err
	}
	var internalEntry timescaleEntry
	err = json.Unmarshal(intermediate, &internalEntry) // nolint
	if err != nil {
		return nil, err
	}

	internalEntry.RequestId = e.RequestId
	if e.Error != nil {
		internalEntry.Error = e.Error.Error()
	}

	return &internalEntry, nil
}

func (_ *timescaleAuditing) toExternal(e timescaleEntry) (Entry, error) {
	intermediate, err := json.Marshal(e) // nolint
	if err != nil {
		return Entry{}, err
	}
	var externalEntry Entry
	err = json.Unmarshal(intermediate, &externalEntry) // nolint
	if err != nil {
		return Entry{}, err
	}

	externalEntry.Id = e.RequestId
	if e.Error != "" {
		externalEntry.Error = errors.New(e.Error)
	}

	return externalEntry, nil
}