package dojo

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// Demotion is a reviewed decision that a golden package no longer is: a person found it does
// not meet the contract the judge is asked about. Promotions are append-only, so a demotion
// is its own record, bound to the content digest like the promotion it withdraws. A later
// promotion of the same digest makes it golden again.
type Demotion struct {
	Digest     string `json:"digest"`
	Package    string `json:"package"`
	ReviewedBy string `json:"reviewed_by"`
	Reason     string `json:"reason"`
	At         string `json:"at"`
}

const demotedFile = "demoted.jsonl"

// acceptDemotion withdraws a promotion made at or before the demotion.
func (l Labels) acceptDemotion(raw []byte) bool {
	var d Demotion
	if json.Unmarshal(raw, &d) != nil || d.Digest == "" || d.ReviewedBy == "" || d.Reason == "" {
		return false
	}
	if promotion, golden := l.Golden[d.Digest]; golden && promotion.At <= d.At {
		delete(l.Golden, d.Digest)
	}
	l.Demoted[d.Digest] = d
	return true
}

// AppendDemotion records that a currently golden package is withdrawn from the golden set. A
// reviewer and a reason are required, and only a golden digest can be demoted.
func AppendDemotion(basePath string, d Demotion) error {
	if strings.TrimSpace(d.ReviewedBy) == "" || strings.TrimSpace(d.Reason) == "" || d.Digest == "" {
		return errors.New("dojo: a demotion needs a digest, a reviewer and a reason")
	}
	labels, err := LoadLabels(basePath)
	if err != nil {
		return err
	}
	if _, golden := labels.Golden[d.Digest]; !golden {
		return fmt.Errorf("dojo: digest %s is not golden, so there is nothing to demote", d.Digest)
	}
	return appendJSONL(filepath.Join(basePath, "dojo", DirCorpus, demotedFile), d)
}
