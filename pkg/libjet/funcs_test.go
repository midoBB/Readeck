// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package libjet

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIndirect(t *testing.T) {
	sV := "test"
	sB := true
	sI := 12
	var pV *string

	type fooType struct {
		v int
	}
	var pF *fooType
	sF := fooType{}

	values := []struct {
		v      interface{}
		expect interface{}
		isNil  bool
	}{
		{sV, "test", false},
		{&sV, "test", false},
		{pV, nil, true},
		{sB, true, false},
		{&sB, true, false},
		{sI, 12, false},
		{&sI, 12, false},
		{fooType{}, fooType{}, false},
		{pF, nil, true},
		{sF, fooType{}, false},
		{fooType{2}, fooType{2}, false},
	}

	for _, tt := range values {
		r, isNil := Indirect(reflect.ValueOf(tt.v))
		require.Exactly(t, tt.expect, r, "%#v", tt.v)
		require.Equal(t, tt.isNil, isNil, "%#v", tt.v)
	}
}

func TestToString(t *testing.T) {
	sV := "test"
	sB := true
	sI := 12
	var pV *string

	type fooType struct {
		v int
	}
	var pF *fooType
	sF := fooType{}

	values := []struct {
		v      interface{}
		expect string
	}{
		{sV, "test"},
		{&sV, "test"},
		{pV, ""},
		{sB, "true"},
		{&sB, "true"},
		{sI, "12"},
		{&sI, "12"},
		{45.5, "45.5"},
		{fooType{}, "{0}"},
		{pF, ""},
		{sF, "{0}"},
		{fooType{2}, "{2}"},
		{[]byte("test"), "test"},
	}

	for _, tt := range values {
		v := ToString(reflect.ValueOf(tt.v))
		require.Equal(t, tt.expect, v, "%#v", tt.v)
	}
}
