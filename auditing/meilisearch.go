package auditing

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/meilisearch/meilisearch-go"
	"go.uber.org/zap"
)

type meiliAuditing struct {
	component        string
	client           *meilisearch.Client
	log              *zap.SugaredLogger
	indexPrefix      string
	rotationInterval Interval
	keep             int64

	indexLock sync.Mutex
	index     *meilisearch.Index
}

const meiliIndexNameTimeSuffixSchema = "\\d\\d\\d\\d-\\d\\d(-\\d\\d(_\\d\\d)?)?"

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
	index, err := a.getLatestIndex()
	if err != nil {
		return err
	}
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

	task, err := index.AddDocuments(documents, "id")
	if err != nil {
		a.log.Errorw("index", "error", err)
		return err
	}
	a.log.Debugw("index", "task", task.TaskUID, "index", index.UID)
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

	if !filter.From.IsZero() {
		predicates = append(predicates, fmt.Sprintf("timestamp-unix >= %d", filter.From.Unix()))
	}
	if !filter.To.IsZero() {
		predicates = append(predicates, fmt.Sprintf("timestamp-unix <= %d", filter.To.Unix()))
	}

	if filter.Limit == 0 {
		filter.Limit = EntryFilterDefaultLimit
	}

	reqProto := meilisearch.SearchRequest{
		Filter:           predicates,
		Query:            filter.Body,
		Sort:             []string{"timestamp-unix:desc", "sort-weight:desc"},
		Limit:            filter.Limit,
		MatchingStrategy: "all",
	}
	req := &meilisearch.MultiSearchRequest{
		Queries: []meilisearch.SearchRequest{},
	}

	_, err := a.getLatestIndex()
	if err != nil {
		return nil, err
	}
	indexes, err := a.getAllIndexes()
	if err != nil {
		return nil, err
	}
	if indexes.Total == 0 {
		return nil, nil
	}
	for _, index := range indexes.Results {
		if !isIndexRelevantForSearchRange(index.UID, filter.From, filter.To) {
			continue
		}

		indexQuery := reqProto
		indexQuery.IndexUID = index.UID
		req.Queries = append(req.Queries, indexQuery)

		i := index
		err = a.migrateIndexSettings(&i)
		if err != nil {
			return nil, err
		}
	}

	if len(req.Queries) == 0 {
		return nil, nil
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

func (a *meiliAuditing) getLatestIndex() (*meilisearch.Index, error) {
	a.indexLock.Lock()
	defer a.indexLock.Unlock()
	indexUid := indexName(a.indexPrefix, a.rotationInterval)
	if a.index != nil && a.index.UID == indexUid {
		return a.index, nil
	}

	a.log.Debugw("auditing", "create new index", a.rotationInterval, "index", indexUid)
	creationTask, err := a.client.CreateIndex(&meilisearch.IndexConfig{
		Uid:        indexUid,
		PrimaryKey: "id",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to request create index (%s): %w", indexUid, err)
	}
	_, err = a.client.WaitForTask(creationTask.TaskUID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute create index (%s): %w", indexUid, err)
	}

	a.index = a.client.Index(indexUid)
	err = a.migrateIndexSettings(a.index)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate index settings (%s): %w", indexUid, err)
	}

	go func() {
		err = a.cleanUpIndexes()
		if err != nil {
			a.log.Errorw("auditing", "failed to clean up indexes", err)
		}
	}()
	return a.index, nil
}

func (a *meiliAuditing) migrateIndexSettings(index *meilisearch.Index) error {
	current, err := index.GetSettings()
	if err != nil {
		return fmt.Errorf("failed to request settings for index (%s): %w", index.UID, err)
	}
	changesRequired := false
	desired := &meilisearch.Settings{
		TypoTolerance: &meilisearch.TypoTolerance{
			Enabled: false,
		},
		SortableAttributes: []string{
			"timestamp-unix",
			"sort-weight",
		},
		SearchableAttributes: []string{
			"body",
			"path",
			"error",
		},
		FilterableAttributes: []string{
			"id",
			"component",
			"rqid",
			"type",
			"timestamp-unix",
			"timestamp",
			"user",
			"tenant",
			"detail",
			"phase",
			"path",
			"forwarded-for",
			"remote-addr",
			"body",
			"status-code",
			"error",
		},
	}
	diff := &meilisearch.Settings{}

	// We'd like to disable the typo tolerance completely, but that's not possible, yet.
	// TypoTolerance.Enabled is marked as omitempty and therefore prevents overriding true with false.
	// https://github.com/meilisearch/meilisearch-go/issues/452
	//
	// if current.TypoTolerance != nil && current.TypoTolerance.Enabled != desired.TypoTolerance.Enabled {
	// 	changesRequired = true
	// 	diff.TypoTolerance = desired.TypoTolerance
	// }

	slices.Sort(current.SortableAttributes)
	slices.Sort(desired.SortableAttributes)
	if !slices.Equal(current.SortableAttributes, desired.SortableAttributes) {
		changesRequired = true
		diff.SortableAttributes = desired.SortableAttributes
	}
	slices.Sort(current.SearchableAttributes)
	slices.Sort(desired.SearchableAttributes)
	if !slices.Equal(current.SearchableAttributes, desired.SearchableAttributes) {
		changesRequired = true
		diff.SearchableAttributes = desired.SearchableAttributes
	}
	slices.Sort(current.FilterableAttributes)
	slices.Sort(desired.FilterableAttributes)
	if !slices.Equal(current.FilterableAttributes, desired.FilterableAttributes) {
		changesRequired = true
		diff.FilterableAttributes = desired.FilterableAttributes
	}
	if !changesRequired {
		return nil
	}

	settingsTask, err := index.UpdateSettings(diff)
	if err != nil {
		return fmt.Errorf("failed to request update settings for index (%s): %w", index.UID, err)
	}
	_, err = a.client.WaitForTask(settingsTask.TaskUID)
	if err != nil {
		return fmt.Errorf("failed to execute update settings for index (%s): %w", index.UID, err)
	}
	return nil
}

func (a *meiliAuditing) cleanUpIndexes() error {
	if a.keep == 0 {
		return nil
	}

	indexListResponse, err := a.getAllIndexes()
	if err != nil {
		a.log.Errorw("unable to list indexes", "err", err)
		return err
	}

	a.log.Debugw("indexes listed", "count", indexListResponse.Total, "keep", a.keep)

	// Sort the indexes descending by creation date
	slices.SortStableFunc(indexListResponse.Results, func(a, b meilisearch.Index) int {
		switch {
		case a.CreatedAt.After(b.CreatedAt):
			return -1
		case a.CreatedAt.Before(b.CreatedAt):
			return 1
		default:
			return 0
		}
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
	a.log.Infow("cleanup finished", "deletes", deleted, "keep", a.keep, "errs", len(errs))
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (a *meiliAuditing) getAllIndexes() (*meilisearch.IndexesResults, error) {
	// First get one index to get total amount of indexes
	indexListResponse, err := a.client.GetIndexes(&meilisearch.IndexesQuery{
		Limit: a.keep + 1,
	})
	if err != nil {
		return nil, err
	}
	if indexListResponse.Total <= indexListResponse.Limit {
		return indexListResponse, nil
	}
	// Now get all indexes
	return a.client.GetIndexes(&meilisearch.IndexesQuery{
		Limit: indexListResponse.Total,
	})
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

func isIndexRelevantForSearchRange(indexName string, from, to time.Time) bool {
	intervalRe := regexp.MustCompile(meiliIndexNameTimeSuffixSchema)
	interval := intervalRe.FindString(indexName)
	formats := map[Interval]string{
		HourlyInterval:  "2006-01-02_15",
		DailyInterval:   "2006-01-02",
		MonthlyInterval: "2006-01",
	}
	for inter, layout := range formats {
		start, err := time.Parse(layout, interval)
		if err != nil {
			continue
		}

		if start.Equal(from) || start.Equal(to) {
			return true
		}

		var end time.Time
		switch inter {
		case HourlyInterval:
			end = start.Add(time.Hour).Add(-time.Nanosecond)
		case DailyInterval:
			end = start.AddDate(0, 0, 1).Add(-time.Nanosecond)
		case MonthlyInterval:
			end = start.AddDate(0, 1, 0).Add(-time.Nanosecond)
		}

		if end.Equal(from) || end.Equal(to) {
			return true
		}
		if from.IsZero() && (start.Before(to) || end.Before(to)) {
			return true
		}
		if to.IsZero() && (start.After(from) || end.After(from)) {
			return true
		}

		return !start.After(to) && !end.Before(from)
	}
	return false
}
