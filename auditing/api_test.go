package auditing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEntryFilter_FillDefaults(t *testing.T) {
	before := time.Now()

	f := EntryFilter{}
	f.FillDefaults()

	assert.False(t, f.From.IsZero(), "From must be filled")
	assert.WithinDuration(t, before.Add(-DefaultSearchWindow), f.From, time.Second, "From must be now minus DefaultSearchWindow")
	assert.True(t, f.From.Before(before), "From must not be in the future")

	// an explicit From is left untouched
	explicit := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	g := EntryFilter{From: explicit}
	g.FillDefaults()
	assert.Equal(t, explicit, g.From)
}
