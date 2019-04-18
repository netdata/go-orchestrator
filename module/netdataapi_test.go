package module

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNetdataAPI_chart(t *testing.T) {
	b := &bytes.Buffer{}
	netdataAPI := apiWriter{Writer: b}

	_ = netdataAPI.chart(
		"",
		"id",
		"name",
		"title",
		"units",
		"family",
		"context",
		Line,
		1,
		1,
		Opts{},
		"orchestrator",
		"module",
	)

	assert.Equal(
		t,
		"CHART '.id' 'name' 'title' 'units' 'family' 'context' 'line' '1' '1' '' 'orchestrator' 'module'\n",
		b.String(),
	)
}

func TestNetdataAPI_dimension(t *testing.T) {
	b := &bytes.Buffer{}
	netdataAPI := apiWriter{Writer: b}

	_ = netdataAPI.dimension(
		"id",
		"name",
		Absolute,
		1,
		1,
		DimOpts{},
	)

	assert.Equal(
		t,
		"DIMENSION 'id' 'name' 'absolute' '1' '1' ''\n",
		b.String(),
	)
}

func TestNetdataAPI_begin(t *testing.T) {
	b := &bytes.Buffer{}
	netdataAPI := apiWriter{Writer: b}

	_ = netdataAPI.begin(
		"typeID",
		"id",
		0,
	)

	assert.Equal(
		t,
		"BEGIN typeID.id\n",
		b.String(),
	)

	b.Reset()

	_ = netdataAPI.begin(
		"typeID",
		"id",
		1,
	)

	assert.Equal(
		t,
		"BEGIN typeID.id 1\n",
		b.String(),
	)
}

func TestNetdataAPI_dimSet(t *testing.T) {
	b := &bytes.Buffer{}
	netdataAPI := apiWriter{Writer: b}

	_ = netdataAPI.dimSet("id", 100)

	assert.Equal(
		t,
		"SET id = 100\n",
		b.String(),
	)
}

func TestNetdataAPI_dimSetEmpty(t *testing.T) {
	b := &bytes.Buffer{}
	netdataAPI := apiWriter{Writer: b}

	_ = netdataAPI.dimSetEmpty("id")

	assert.Equal(
		t,
		"SET id = \n",
		b.String(),
	)
}

func TestNetdataAPI_varSet(t *testing.T) {
	b := &bytes.Buffer{}
	netdataAPI := apiWriter{Writer: b}

	_ = netdataAPI.varSet("id", 100)

	assert.Equal(
		t,
		"VARIABLE CHART id = 100\n",
		b.String(),
	)
}

func TestNetdataAPI_end(t *testing.T) {
	b := &bytes.Buffer{}
	netdataAPI := apiWriter{Writer: b}

	_ = netdataAPI.end()

	assert.Equal(
		t,
		"END\n\n",
		b.String(),
	)
}

func TestNetdataAPI_flush(t *testing.T) {
	b := &bytes.Buffer{}
	netdataAPI := apiWriter{Writer: b}

	_ = netdataAPI.flush()

	assert.Equal(
		t,
		"FLUSH\n",
		b.String(),
	)
}
