package multisort

import (
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
)

func TestSortByWithStrings(t *testing.T) {
	now := time.Now()

	type machine struct {
		ID         string
		Project    string
		Liveliness string
		LastEvent  time.Time
		IP         netip.Addr
	}

	testData := []machine{
		{
			ID:         "004",
			Project:    "B",
			Liveliness: "Alive",
			IP:         netip.MustParseAddr("1.2.3.4"),
			LastEvent:  now.Add(-3 * time.Minute),
		},
		{
			ID:         "001",
			Project:    "B",
			Liveliness: "Unknown",
			IP:         netip.MustParseAddr("1.2.3.1"),
			LastEvent:  now.Add(-2 * time.Minute),
		},
		{
			ID:         "002",
			Project:    "A",
			Liveliness: "Alive",
			IP:         netip.MustParseAddr("1.2.3.2"),
			LastEvent:  now.Add(3 * time.Minute),
		},
		{
			ID:         "003",
			Project:    "A",
			Liveliness: "Unknown",
			IP:         netip.MustParseAddr("1.2.3.3"),
			LastEvent:  now.Add(1 * time.Minute),
		},
	}

	fields := FieldMap[machine]{
		"id": func(a, b machine, descending bool) CompareResult {
			return Compare(a.ID, b.ID, descending)
		},
		"project": func(a, b machine, descending bool) CompareResult {
			return Compare(a.Project, b.Project, descending)
		},
		"liveliness": func(a, b machine, descending bool) CompareResult {
			return Compare(a.Liveliness, b.Liveliness, descending)
		},
		"event": func(a, b machine, descending bool) CompareResult {
			return Compare(a.LastEvent.Unix(), b.LastEvent.Unix(), descending)
		},
		"ip": func(a, b machine, descending bool) CompareResult {
			return WithCompareFunc(func() int {
				return a.IP.Compare(b.IP)
			}, descending)
		},
	}

	tests := []struct {
		name    string
		keys    []Key
		fields  FieldMap[machine]
		data    []machine
		want    []machine
		wantErr error
	}{
		{
			name:   "sort without key does not change order",
			keys:   Keys{},
			fields: fields,
			data:   testData,
			want: []machine{
				{
					ID:         "004",
					Project:    "B",
					Liveliness: "Alive",
					IP:         netip.MustParseAddr("1.2.3.4"),
					LastEvent:  now.Add(-3 * time.Minute),
				},
				{
					ID:         "001",
					Project:    "B",
					Liveliness: "Unknown",
					IP:         netip.MustParseAddr("1.2.3.1"),
					LastEvent:  now.Add(-2 * time.Minute),
				},
				{
					ID:         "002",
					Project:    "A",
					Liveliness: "Alive",
					IP:         netip.MustParseAddr("1.2.3.2"),
					LastEvent:  now.Add(3 * time.Minute),
				},
				{
					ID:         "003",
					Project:    "A",
					Liveliness: "Unknown",
					IP:         netip.MustParseAddr("1.2.3.3"),
					LastEvent:  now.Add(1 * time.Minute),
				},
			},
		},
		{
			name:   "unknown key does not change order",
			keys:   Keys{{ID: "foo"}},
			fields: fields,
			data:   testData,
			want: []machine{
				{
					ID:         "004",
					Project:    "B",
					Liveliness: "Alive",
					IP:         netip.MustParseAddr("1.2.3.4"),
					LastEvent:  now.Add(-3 * time.Minute),
				},
				{
					ID:         "001",
					Project:    "B",
					Liveliness: "Unknown",
					IP:         netip.MustParseAddr("1.2.3.1"),
					LastEvent:  now.Add(-2 * time.Minute),
				},
				{
					ID:         "002",
					Project:    "A",
					Liveliness: "Alive",
					IP:         netip.MustParseAddr("1.2.3.2"),
					LastEvent:  now.Add(3 * time.Minute),
				},
				{
					ID:         "003",
					Project:    "A",
					Liveliness: "Unknown",
					IP:         netip.MustParseAddr("1.2.3.3"),
					LastEvent:  now.Add(1 * time.Minute),
				},
			},
			wantErr: errors.New("sort key does not exist: foo"),
		},
		{
			name:   "sort by id",
			keys:   Keys{{ID: "id"}},
			fields: fields,
			data:   testData,
			want: []machine{
				{
					ID:         "001",
					Project:    "B",
					Liveliness: "Unknown",
					IP:         netip.MustParseAddr("1.2.3.1"),
					LastEvent:  now.Add(-2 * time.Minute),
				},
				{
					ID:         "002",
					Project:    "A",
					Liveliness: "Alive",
					IP:         netip.MustParseAddr("1.2.3.2"),
					LastEvent:  now.Add(3 * time.Minute),
				},
				{
					ID:         "003",
					Project:    "A",
					Liveliness: "Unknown",
					IP:         netip.MustParseAddr("1.2.3.3"),
					LastEvent:  now.Add(1 * time.Minute),
				},
				{
					ID:         "004",
					Project:    "B",
					Liveliness: "Alive",
					IP:         netip.MustParseAddr("1.2.3.4"),
					LastEvent:  now.Add(-3 * time.Minute),
				},
			},
		},
		{
			name:   "sort by id descending",
			keys:   Keys{{ID: "id", Descending: true}},
			fields: fields,
			data:   testData,
			want: []machine{
				{
					ID:         "004",
					Project:    "B",
					Liveliness: "Alive",
					IP:         netip.MustParseAddr("1.2.3.4"),
					LastEvent:  now.Add(-3 * time.Minute),
				},
				{
					ID:         "003",
					Project:    "A",
					Liveliness: "Unknown",
					IP:         netip.MustParseAddr("1.2.3.3"),
					LastEvent:  now.Add(1 * time.Minute),
				},
				{
					ID:         "002",
					Project:    "A",
					Liveliness: "Alive",
					IP:         netip.MustParseAddr("1.2.3.2"),
					LastEvent:  now.Add(3 * time.Minute),
				},
				{
					ID:         "001",
					Project:    "B",
					Liveliness: "Unknown",
					IP:         netip.MustParseAddr("1.2.3.1"),
					LastEvent:  now.Add(-2 * time.Minute),
				},
			},
		},
		{
			name:   "sort by last project, id",
			keys:   Keys{{ID: "project"}, {ID: "id"}},
			fields: fields,
			data:   testData,
			want: []machine{
				{
					ID:         "002",
					Project:    "A",
					Liveliness: "Alive",
					IP:         netip.MustParseAddr("1.2.3.2"),
					LastEvent:  now.Add(3 * time.Minute),
				},
				{
					ID:         "003",
					Project:    "A",
					Liveliness: "Unknown",
					IP:         netip.MustParseAddr("1.2.3.3"),
					LastEvent:  now.Add(1 * time.Minute),
				},
				{
					ID:         "001",
					Project:    "B",
					Liveliness: "Unknown",
					IP:         netip.MustParseAddr("1.2.3.1"),
					LastEvent:  now.Add(-2 * time.Minute),
				},
				{
					ID:         "004",
					Project:    "B",
					Liveliness: "Alive",
					IP:         netip.MustParseAddr("1.2.3.4"),
					LastEvent:  now.Add(-3 * time.Minute),
				},
			},
		},
		{
			name:   "sort by last event time",
			keys:   Keys{{ID: "event"}},
			fields: fields,
			data:   testData,
			want: []machine{
				{
					ID:         "004",
					Project:    "B",
					Liveliness: "Alive",
					IP:         netip.MustParseAddr("1.2.3.4"),
					LastEvent:  now.Add(-3 * time.Minute),
				},
				{
					ID:         "001",
					Project:    "B",
					Liveliness: "Unknown",
					IP:         netip.MustParseAddr("1.2.3.1"),
					LastEvent:  now.Add(-2 * time.Minute),
				},
				{
					ID:         "003",
					Project:    "A",
					Liveliness: "Unknown",
					IP:         netip.MustParseAddr("1.2.3.3"),
					LastEvent:  now.Add(1 * time.Minute),
				},
				{
					ID:         "002",
					Project:    "A",
					Liveliness: "Alive",
					IP:         netip.MustParseAddr("1.2.3.2"),
					LastEvent:  now.Add(3 * time.Minute),
				},
			},
		},

		{
			name:   "sort by ip",
			keys:   Keys{{ID: "ip"}},
			fields: fields,
			data:   testData,
			want: []machine{
				{
					ID:         "001",
					Project:    "B",
					Liveliness: "Unknown",
					IP:         netip.MustParseAddr("1.2.3.1"),
					LastEvent:  now.Add(-2 * time.Minute),
				},
				{
					ID:         "002",
					Project:    "A",
					Liveliness: "Alive",
					IP:         netip.MustParseAddr("1.2.3.2"),
					LastEvent:  now.Add(3 * time.Minute),
				},
				{
					ID:         "003",
					Project:    "A",
					Liveliness: "Unknown",
					IP:         netip.MustParseAddr("1.2.3.3"),
					LastEvent:  now.Add(1 * time.Minute),
				},
				{
					ID:         "004",
					Project:    "B",
					Liveliness: "Alive",
					IP:         netip.MustParseAddr("1.2.3.4"),
					LastEvent:  now.Add(-3 * time.Minute),
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			sorter := New(tt.fields)
			err := sorter.SortBy(tt.data, tt.keys...)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}

			if diff := cmp.Diff(tt.want, tt.data, testcommon.IgnoreUnexported()); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}
