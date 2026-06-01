package parser

import (
	"reflect"
	"testing"
)

func TestParseHosts(t *testing.T) {
	tests := []struct {
		match string
		want  []string
	}{
		{"`Host(`foo.example.com`)`", []string{"foo.example.com"}},
		{"Host(`foo.example.com`) && PathPrefix(`/api`)", []string{"foo.example.com"}},
		{"Host(`a.com`, `b.com`)", []string{"a.com", "b.com"}},
		{"PathPrefix(`/only`)", nil},
		{"", nil},
	}
	for _, tt := range tests {
		got := ParseHosts(tt.match)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("ParseHosts(%q) = %v, want %v", tt.match, got, tt.want)
		}
	}
}
