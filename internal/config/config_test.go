package config

import (
	"reflect"
	"testing"
)

func TestParseCORSAllowedOrigins(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{name: "empty disables cross-origin access"},
		{
			name:  "trim and deduplicate origins",
			input: " https://erp.example.com, http://localhost:5173,https://erp.example.com ",
			want:  []string{"https://erp.example.com", "http://localhost:5173"},
		},
		{name: "reject wildcard", input: "*", wantErr: true},
		{name: "reject path", input: "https://erp.example.com/app", wantErr: true},
		{name: "reject non-http scheme", input: "file://local", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseCORSAllowedOrigins(test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("parseCORSAllowedOrigins() error = %v, wantErr %t", err, test.wantErr)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("parseCORSAllowedOrigins() = %#v, want %#v", got, test.want)
			}
		})
	}
}
