// Copyright 2016 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package opalog

import "strconv"
import "strings"

const (
	NULL    = iota
	BOOLEAN = iota
	NUMBER  = iota
	STRING  = iota
	ARRAY   = iota
	OBJECT  = iota
	VAR     = iota
)

// Location records a position in source code
type Location struct {
	File string
	Row  int
	Col  int
}

// NewLocation creates a new instance of a location
func NewLocation(file string, row int, col int) *Location {
	l := Location{File: file, Row: row, Col: col}
	return &l
}

// Term is an argument to a function
type Term struct {
	Value    interface{} // actual value, as represented by Go
	Kind     int         // type of Term: one of the consts defined above
	Name     []byte      // original string representation
	Location *Location   // text location in original source
}

// NewTerm creates a new Term
func NewTerm(x interface{}, kind int, orig []byte, file string, row int, col int) *Term {
	t := Term{Value: x, Kind: kind, Name: orig, Location: NewLocation(file, row, col)}
	return &t
}

// String returns the string representation of the Term.
func (t *Term) String() string {
	switch t.Kind {
	case NULL:
		return "null"
	case BOOLEAN:
		return strconv.FormatBool(t.Value.(bool))
	case NUMBER:
		return strconv.FormatFloat(t.Value.(float64), 'G', -1, 64)
	case STRING:
		return "\"" + t.Value.(string) + "\""
	case VAR:
		return t.Value.(Var).Name
	case ARRAY:
		var buf []string
		for _, v := range t.Value.([]*Term) {
			buf = append(buf, v.String())
		}
		return "[" + strings.Join(buf, ", ") + "]"
	case OBJECT:
		set := t.Value.(*Set)
		var buf []string
		for _, v := range set.Values {
			buf = append(buf, v.(*KeyValue).String())
		}
		return "{" + strings.Join(buf, ", ") + "}"
	}
	panic("unreachable")
	return ""
}

// Equal checks if two terms are equal for their Value and Kind fields.
// Ignores differences in pointers.
// Will infinite loop on circular Terms (which are never generated by the parser).
func (term1 *Term) Equal(term2 *Term) bool {
	// pointer equality
	if term1 == term2 {
		return true
	}
	// wrong types
	if term1.Kind != term2.Kind {
		return false
	}
	// recursive cases
	switch term1.Kind {
	case OBJECT:
		// A dictionary is a list of key/value pairs because
		//   the keys may not be simple strings in the language
		set1 := term1.Value.(*Set)
		set2 := term2.Value.(*Set)
		return set1.Equal(set2)
	case ARRAY:
		// Golang Value objs for each of the Terms' .Value fields
		arr1 := term1.Value.([]*Term)
		arr2 := term2.Value.([]*Term)
		if len(arr1) != len(arr2) {
			return false
		}
		for i := 0; i < len(arr1); i++ {
			if !arr1[i].Equal(arr2[i]) {
				return false
			}
		}
		return true
	case VAR:
		var1 := term1.Value.(*Var)
		var2 := term2.Value.(*Var)
		return var1.Name == var2.Name
	default:
		return term1.Value == term2.Value
	}
}