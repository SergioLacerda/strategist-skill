package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortItemRows_TieBreaksByKindThenID(t *testing.T) {
	rows := []itemRow{
		{ChestID: "b", Kind: "jewel", ID: "z"},
		{ChestID: "a", Kind: "potion", ID: "m"},
		{ChestID: "a", Kind: "jewel", ID: "z"},
		{ChestID: "a", Kind: "jewel", ID: "a"},
	}
	sortItemRows(rows)

	got := make([]string, len(rows))
	for i, r := range rows {
		got[i] = r.ChestID + ":" + r.Kind + ":" + r.ID
	}
	assert.Equal(t, []string{"a:jewel:a", "a:jewel:z", "a:potion:m", "b:jewel:z"}, got)
}
