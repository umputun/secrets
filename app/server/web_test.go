package server

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestTemplates_Duration(t *testing.T) {
	tests := []struct {
		name string
		unit string
		v    int
		want time.Duration
	}{
		{
			name: "minutes",
			unit: "m",
			v:    5,
			want: time.Duration(5) * time.Minute,
		},
		{
			name: "hours",
			unit: "h",
			v:    5,
			want: time.Duration(5) * time.Hour,
		},
		{
			name: "days",
			unit: "d",
			v:    5,
			want: time.Duration(5*24) * time.Hour,
		},
		{
			name: "bad unit",
			unit: "x",
			v:    5,
			want: time.Duration(0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := duration(tt.v, tt.unit)
			t.Logf("%+v", r)

			assert.Equal(t, tt.want, r)
		})
	}
}

func TestTemplates_HumanDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{
			name: "seconds",
			d:    time.Duration(5) * time.Second,
			want: "5 seconds",
		},
		{
			name: "minutes",
			d:    time.Duration(5) * time.Minute,
			want: "5 minutes",
		},
		{
			name: "hours",
			d:    time.Duration(5) * time.Hour,
			want: "5 hours",
		},
		{
			name: "days",
			d:    time.Duration(5*24) * time.Hour,
			want: "5 days",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := humanDuration(tt.d)
			t.Logf("%+v", r)

			assert.Equal(t, tt.want, r)
		})
	}
}

func TestTemplates_NewTemplateCache(t *testing.T) {
	cache, err := newTemplateCache()

	assert.NoError(t, err)

	assert.Equal(t, 8, len(cache))
	assert.NotNil(t, cache["404.tmpl.html"])
	assert.NotNil(t, cache["about.tmpl.html"])
	assert.NotNil(t, cache["home.tmpl.html"])
	assert.NotNil(t, cache["show-message.tmpl.html"])
	assert.NotNil(t, cache["decoded-message.tmpl.html"])
	assert.NotNil(t, cache["error.tmpl.html"])
	assert.NotNil(t, cache["footer.tmpl.html"])
	assert.NotNil(t, cache["secure-link.tmpl.html"])
}
