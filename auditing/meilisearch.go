package auditing

import (
	"errors"
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
	if c.Component == "" {
		ex, err := os.Executable()
		if err != nil {
			return nil, err
		}
		c.Component = filepath.Base(ex)
	}

	client := meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   c.URL,
		APIKey: c.APIKey,
	})
	v, err := client.GetVersion()
	if err != nil {
		return nil, fmt.Errorf("unable to connect to meilisearch at:%s %w", c.URL, err)
	}
	c.Log.Infow("meilisearch", "connected to", v, "index rotated", c.RotationInterval, "index keep", c.Keep)

	a := &meiliAuditing{
		component:        c.Component,
		client:           client,
		log:              c.Log.Named("auditing"),
		indexPrefix:      c.IndexPrefix,
		rotationInterval: c.RotationInterval,
		keep:             c.Keep,
	}
	err = a.newIndex()
	if err != nil {
		return nil, err
	}

	if c.RotationInterval != "" {
		// create a new Index every interval
		cn := cron.New()
		err := cn.AddFunc(string(c.RotationInterval), func() {
			err := a.newIndex()
			if err != nil {
				a.log.Errorw("index rotation", "error", err)
			}
		})
		if err != nil {
			return nil, err
		}
		cn.Start()
	}
	return a, nil
}

func (a *meiliAuditing) Flush() error {
	taskResult, err := a.client.GetTasks(&meilisearch.TasksQuery{Statuses: []string{"enqueued", "processing"}, Limit: 100})
	if err != nil {
		return err
	}
	a.log.Debugw("flush, waiting for", "tasks", len(taskResult.Results))

	var errs []error
	for _, task := range taskResult.Results {
		_, err := a.client.WaitForTask(task.UID)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
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

	doc := a.encodeEntry(entry)
	documents := []map[string]any{doc}

	task, err := a.index.AddDocuments(documents, "id")
	if err != nil {
		a.log.Errorw("index", "error", err)
		return err
	}
	a.log.Debugw("index", "task", task.TaskUID, "index", a.index.UID)

	stats, _ := a.index.GetStats()
	a.log.Debugw("index", "task status", task.Status, "index stats", stats)
	return nil
}

func (a *meiliAuditing) Search(filter EntryFilter) ([]Entry, error) {
	predicates := make([]string, 0)
	if filter.Component != "" {
		predicates = append(predicates, fmt.Sprintf("component = %q", filter.Component))
	}
	if filter.Type != "" {
		predicates = append(predicates, fmt.Sprintf("type = %q", filter.Type))
	}
	if filter.User != "" {
		predicates = append(predicates, fmt.Sprintf("user = %q", filter.User))
	}
	if filter.Tenant != "" {
		predicates = append(predicates, fmt.Sprintf("tenant = %q", filter.Tenant))
	}
	if filter.RequestId != "" {
		predicates = append(predicates, fmt.Sprintf("rqid = %q", filter.RequestId))
	}
	if filter.Detail != "" {
		predicates = append(predicates, fmt.Sprintf("detail = %q", filter.Detail))
	}
	if filter.Phase != "" {
		predicates = append(predicates, fmt.Sprintf("phase = %q", filter.Phase))
	}
	if filter.Path != "" {
		predicates = append(predicates, fmt.Sprintf("path = %q", filter.Path))
	}
	if filter.ForwardedFor != "" {
		predicates = append(predicates, fmt.Sprintf("forwarded-for = %q", filter.ForwardedFor))
	}
	if filter.RemoteAddr != "" {
		predicates = append(predicates, fmt.Sprintf("remote-addr = %q", filter.RemoteAddr))
	}
	if filter.StatusCode != 0 {
		predicates = append(predicates, fmt.Sprintf("status-code = %d", filter.StatusCode))
	}
	if filter.Error != "" {
		predicates = append(predicates, fmt.Sprintf("error = %q", filter.Error))
	}

	if filter.From.IsZero() {
		filter.From = time.Now().Add(-time.Hour)
	}
	predicates = append(predicates, fmt.Sprintf("timestamp-unix >= %d", filter.From.Unix()))
	if filter.To.IsZero() {
		filter.To = time.Now()
	}
	predicates = append(predicates, fmt.Sprintf("timestamp-unix <= %d", filter.To.Unix()))

	if filter.Limit == 0 {
		filter.Limit = 100
	}

	reqProto := meilisearch.SearchRequest{
		Filter: predicates,
		Query:  filter.Body,
		Sort:   []string{"timestamp-unix:desc", "sort-weight:desc"},
		Limit:  filter.Limit,
	}
	req := &meilisearch.MultiSearchRequest{}

	indexes, err := a.client.GetIndexes(&meilisearch.IndexesQuery{})
	if err != nil {
		return nil, err
	}
	for _, index := range indexes.Results {
		if !strings.HasPrefix(index.UID, a.indexPrefix) {
			continue
		}
		indexQuery := reqProto
		indexQuery.IndexUID = index.UID
		req.Queries = append(req.Queries, indexQuery)
	}

	resp, err := a.client.MultiSearch(req)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0)
	for _, r := range resp.Results {
		for _, h := range r.Hits {
			h, ok := h.(map[string]any)
			if !ok {
				continue
			}
			entries = append(entries, a.decodeEntry(h))
		}
	}
	return entries, nil
}

func (a *meiliAuditing) encodeEntry(entry Entry) map[string]any {
	doc := make(map[string]any)
	doc["id"] = entry.Id
	doc["component"] = entry.Component
	doc["sort-weight"] = a.entrySortWeight(entry)
	if entry.Type != "" {
		doc["type"] = string(entry.Type)
	}
	doc["timestamp"] = entry.Timestamp.Format(time.RFC3339)
	doc["timestamp-unix"] = entry.Timestamp.Unix()
	if entry.User != "" {
		doc["user"] = entry.User
	}
	if entry.Tenant != "" {
		doc["tenant"] = entry.Tenant
	}
	if entry.RequestId != "" {
		doc["rqid"] = entry.RequestId
	}
	if entry.Detail != "" {
		doc["detail"] = string(entry.Detail)
	}
	if entry.Phase != "" {
		doc["phase"] = string(entry.Phase)
	}
	if entry.Path != "" {
		doc["path"] = entry.Path
	}
	if entry.ForwardedFor != "" {
		doc["forwarded-for"] = entry.ForwardedFor
	}
	if entry.RemoteAddr != "" {
		doc["remote-addr"] = entry.RemoteAddr
	}
	if entry.StatusCode != 0 {
		doc["status-code"] = entry.StatusCode
	}
	if entry.Error != nil {
		doc["error"] = entry.Error.Error()
	}
	if entry.Body != nil {
		doc["body"] = entry.Body
	}
	return doc
}

func (a *meiliAuditing) entrySortWeight(entry Entry) float32 {
	switch entry.Phase {
	case EntryPhaseOpened:
		return 1
	case EntryPhaseRequest:
		return 2
	case EntryPhaseSingle:
		return 3
	case EntryPhaseResponse:
		return 4
	case EntryPhaseError:
		return 5
	case EntryPhaseClosed:
		return 6
	default:
		return 0
	}
}

func (a *meiliAuditing) decodeEntry(doc map[string]any) Entry {
	var entry Entry
	if id, ok := doc["id"].(string); ok {
		entry.Id = id
	}
	if component, ok := doc["component"].(string); ok {
		entry.Component = component
	}
	if t, ok := doc["type"].(string); ok {
		entry.Type = EntryType(t)
	}
	if timestampUnix, ok := doc["timestamp-unix"].(int64); ok {
		entry.Timestamp = time.Unix(timestampUnix, 0)
	}
	if timestamp, ok := doc["timestamp"].(string); ok {
		entry.Timestamp, _ = time.Parse(time.RFC3339, timestamp)
	}
	if user, ok := doc["user"].(string); ok {
		entry.User = user
	}
	if tenant, ok := doc["tenant"].(string); ok {
		entry.Tenant = tenant
	}
	if rqid, ok := doc["rqid"].(string); ok {
		entry.RequestId = rqid
	}
	if detail, ok := doc["detail"].(string); ok {
		entry.Detail = EntryDetail(detail)
	}
	if phase, ok := doc["phase"].(string); ok {
		entry.Phase = EntryPhase(phase)
	}
	if path, ok := doc["path"].(string); ok {
		entry.Path = path
	}
	if forwardedFor, ok := doc["forwarded-for"].(string); ok {
		entry.ForwardedFor = forwardedFor
	}
	if remoteAddr, ok := doc["remote-addr"].(string); ok {
		entry.RemoteAddr = remoteAddr
	}
	if statusCode, ok := doc["status-code"].(float64); ok {
		entry.StatusCode = int(statusCode)
	}
	if err, ok := doc["error"].(string); ok {
		entry.Error = errors.New(err)
	}
	if body, ok := doc["body"]; ok {
		entry.Body = body
	}
	return entry

}

func (a *meiliAuditing) newIndex() error {
	a.log.Debugw("auditing", "create new index", a.rotationInterval)
	a.index = a.client.Index(indexName(a.indexPrefix, a.rotationInterval))

	tTypo, err := a.index.UpdateTypoTolerance(&meilisearch.TypoTolerance{
		Enabled: false,
	})
	if err != nil {
		return err
	}
	tSort, err := a.index.UpdateSortableAttributes(&[]string{"timestamp-unix", "sort-weight"})
	if err != nil {
		return err
	}
	tFilter, err := a.index.UpdateFilterableAttributes(&[]string{"component", "type", "user", "tenant", "rqid", "detail", "phase", "path", "forwarded-for", "remote-addr", "timestamp-unix", "body"})
	if err != nil {
		return err
	}
	var errs []error
	for _, task := range []*meilisearch.TaskInfo{tTypo, tSort, tFilter} {
		_, err = a.client.WaitForTask(task.TaskUID)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		a.log.Errorw("unable to update index settings", "err", errs)
		return errors.Join(errs...)
	}
	return a.cleanUpIndexes()
}

func (a *meiliAuditing) cleanUpIndexes() error {
	if a.keep == 0 {
		return nil
	}
	// First get one index to get total amount of indexes
	indexListResponse, err := a.client.GetIndexes(&meilisearch.IndexesQuery{
		Limit: 1,
	})
	if err != nil {
		a.log.Errorw("unable to list indexes", "err", err)
		return err
	}
	// Now get all indexes
	indexListResponse, err = a.client.GetIndexes(&meilisearch.IndexesQuery{
		Limit: indexListResponse.Total,
	})
	if err != nil {
		a.log.Errorw("unable to list indexes", "err", err)
		return err
	}

	a.log.Debugw("indexes listed", "count", indexListResponse.Total, "keep", a.keep)

	// Sort the indexes descending by creation date
	slices.SortStableFunc(indexListResponse.Results, func(a, b meilisearch.Index) bool {
		return a.CreatedAt.After(b.CreatedAt)
	})

	deleted := 0
	seen := 0
	var errs []error
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
			errs = append(errs, err)
			continue
		}
		deleted++
		a.log.Debugw("deleted index", "uid", index.UID, "created", index.CreatedAt, "info", deleteInfo)
	}
	a.log.Debugw("done deleting indexes", "deleted", deleted)
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
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
