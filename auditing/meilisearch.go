package auditing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/meilisearch/meilisearch-go"
	"github.com/robfig/cron"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

type meiliAuditing struct {
	component        string
	client           *meilisearch.Client
	index            *meilisearch.Index
	log              *zap.SugaredLogger
	indexPrefix      string
	rotationInterval Interval
	keep             int64
}

func New(c Config) (Auditing, error) {
	client := meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   c.URL,
		APIKey: c.APIKey,
	})
	v, err := client.GetVersion()
	if err != nil {
		return nil, fmt.Errorf("unable to connect to meilisearch at:%s %w", c.URL, err)
	}
	c.Log.Infow("meilisearch", "connected to", v, "index rotated", c.RotationInterval, "index keep", c.Keep)

	index := client.Index(c.IndexPrefix)
	if c.RotationInterval != "" {
		index = client.Index(indexName(c.IndexPrefix, c.RotationInterval))
	}

	a := &meiliAuditing{
		component:        c.Component,
		client:           client,
		index:            index,
		log:              c.Log.Named("auditing"),
		indexPrefix:      c.IndexPrefix,
		rotationInterval: c.RotationInterval,
		keep:             c.Keep,
	}
	if a.component == "" {
		ex, err := os.Executable()
		if err != nil {
			return nil, err
		}
		a.component = filepath.Base(ex)
	}

	if c.RotationInterval != "" {
		// create a new Index every interval
		cn := cron.New()
		err := cn.AddFunc(string(c.RotationInterval), a.newIndex)
		if err != nil {
			return nil, err
		}
		cn.Start()
	}
	return a, nil
}

func (a *meiliAuditing) Index(entry Entry) error {
	if entry.Id == "" {
		entry.Id = uuid.NewString()
	}
	if entry.Component == "" {
		entry.Component = a.component
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	documents := []Entry{entry}

	task, err := a.index.AddDocuments(documents)
	if err != nil {
		a.log.Errorw("index", "error", err)
		return err
	}
	stats, _ := a.index.GetStats()
	a.log.Debugw("index", "task status", task.Status, "index stats", stats)
	return nil
}

func (a *meiliAuditing) newIndex() {
	a.log.Debugw("auditing", "create new index", a.rotationInterval)
	a.index = a.client.Index(indexName(a.indexPrefix, a.rotationInterval))
	a.cleanUpIndexes()
}

func (a *meiliAuditing) cleanUpIndexes() {
	if a.keep == 0 {
		return
	}
	// First get one index to get total amount of indexes
	indexListResponse, err := a.client.GetIndexes(&meilisearch.IndexesQuery{
		Limit: 1,
	})
	if err != nil {
		a.log.Errorw("unable to list indexes")
		return
	}
	// Now get all indexes
	indexListResponse, err = a.client.GetIndexes(&meilisearch.IndexesQuery{
		Limit: indexListResponse.Total,
	})
	if err != nil {
		a.log.Errorw("unable to list indexes")
		return
	}

	a.log.Debugw("indexes listed", "count", indexListResponse.Total, "keep", a.keep)

	// Sort the indexes descending by creation date
	slices.SortStableFunc(indexListResponse.Results, func(a, b meilisearch.Index) bool {
		return a.CreatedAt.After(b.CreatedAt)
	})

	deleted := 0
	seen := 0
	for _, index := range indexListResponse.Results {
		a.log.Debugw("inspect index for deletion", "uid", index.UID)
		if !strings.HasPrefix(index.UID, a.indexPrefix) {
			continue
		}
		seen++
		if seen < int(a.keep) {
			continue
		}
		deleteInfo, err := a.client.DeleteIndex(index.UID)
		if err != nil {
			a.log.Errorw("unable to delete index", "uid", index.UID, "created", index.CreatedAt)
			continue
		}
		deleted++
		a.log.Debugw("deleted index", "uid", index.UID, "created", index.CreatedAt, "info", deleteInfo)
	}
	a.log.Debugw("done deleting indexes", "deleted", deleted)
}

func indexName(prefix string, i Interval) string {
	timeFormat := "2006-01-02"

	switch i {
	case HourlyInterval:
		timeFormat = "2006-01-02_15"
	case DailyInterval:
		timeFormat = "2006-01-02"
	case MonthlyInterval:
		timeFormat = "2006-01"
	}

	indexName := prefix + "-" + time.Now().Format(timeFormat)
	return indexName
}
