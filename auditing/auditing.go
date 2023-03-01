package auditing

import "go.uber.org/zap"

type Config struct {
	Component        string
	URL              string
	APIKey           string
	IndexPrefix      string
	RotationInterval Interval
	Keep             int64
	Log              *zap.SugaredLogger
}

type Interval string

var (
	HourlyInterval  Interval = "@hourly"
	DailyInterval   Interval = "@daily"
	MonthlyInterval Interval = "@monthly"
)

type Auditing interface {
	Index(...any) error
}
