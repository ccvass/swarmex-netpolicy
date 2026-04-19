package netpolicy

import (
	"reflect"
	"testing"
)

func TestParsePolicy(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   Policy
	}{
		{
			"empty",
			map[string]string{},
			Policy{},
		},
		{
			"deny all",
			map[string]string{labelDenyAll: "true"},
			Policy{DenyAll: true},
		},
		{
			"ingress only",
			map[string]string{labelIngress: "web,api"},
			Policy{Ingress: []string{"web", "api"}},
		},
		{
			"full policy",
			map[string]string{
				labelDenyAll: "true",
				labelIngress: "frontend",
				labelEgress:  "db, cache",
			},
			Policy{DenyAll: true, Ingress: []string{"frontend"}, Egress: []string{"db", "cache"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParsePolicy(tt.labels)
			if got.DenyAll != tt.want.DenyAll ||
				!reflect.DeepEqual(got.Ingress, tt.want.Ingress) ||
				!reflect.DeepEqual(got.Egress, tt.want.Egress) {
				t.Errorf("ParsePolicy() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"web", []string{"web"}},
		{"web, api, db", []string{"web", "api", "db"}},
		{" , ,", nil},
	}
	for _, tt := range tests {
		got := splitCSV(tt.input)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("splitCSV(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestStripCIDR(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"10.0.0.5/24", "10.0.0.5"},
		{"10.0.0.5", "10.0.0.5"},
	}
	for _, tt := range tests {
		if got := stripCIDR(tt.input); got != tt.want {
			t.Errorf("stripCIDR(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
