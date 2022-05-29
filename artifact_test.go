package artifact

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Denyer struct {
	Notifier

	entries map[string]struct{}
}

var _ Artifact = (*Denyer)(nil)

func (d *Denyer) Update(in io.Reader) error {
	var entries []string
	if err := json.NewDecoder(in).Decode(&entries); err != nil {
		return err
	}
	for _, entry := range entries {
		d.entries[entry] = struct{}{}
	}
	return nil
}

func TestUpdates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		scenario string
		input    io.Reader
		values   map[string]struct{}
		err      error
	}{
		{
			scenario: "Simple list of entries",
			input:    strings.NewReader(`["foo", "bar"]`),
			values: map[string]struct{}{
				"foo": {}, "bar": {},
			},
			err: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.scenario, func(t *testing.T) {
			d := &Denyer{
				entries: map[string]struct{}{},
			}
			assert.ErrorIs(t, d.Update(tc.input), tc.err, "Must match the expected error")
			assert.Equal(t, tc.values, d.entries, "Must match the expected entries")
		})
	}
}
