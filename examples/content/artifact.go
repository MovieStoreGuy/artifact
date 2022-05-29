package main

import (
	"encoding/json"
	"io"

	"github.com/MovieStoreGuy/artifact"
)

type Content struct {
	artifact.Notifier

	Headlines []string `json:"headlines"`
	Links     []string `json:"links"`
}

var _ artifact.Artifact = (*Content)(nil)

func (c *Content) Update(in io.Reader) error {
	return json.NewDecoder(in).Decode(c)
}
