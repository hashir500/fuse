package cmd

import "testing"

func TestPercent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value float64
		max   float64
		want  string
	}{
		{name: "zero max", value: 1, max: 0, want: "0%"},
		{name: "less than point one percent", value: 0.00007, max: 100, want: "<0.1%"},
		{name: "micro spend", value: 0.000011, max: 0.001, want: "1.1%"},
		{name: "normal spend", value: 12, max: 100, want: "12%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := percent(tt.value, tt.max); got != tt.want {
				t.Fatalf("percent(%f, %f) = %q, want %q", tt.value, tt.max, got, tt.want)
			}
		})
	}
}

func TestBar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value float64
		max   float64
		want  string
	}{
		{name: "zero", value: 0, max: 100, want: ".........."},
		{name: "five percent", value: 5, max: 100, want: "#........."},
		{name: "fifty percent", value: 50, max: 100, want: "#####....."},
		{name: "ninety five percent", value: 95, max: 100, want: "##########"},
		{name: "over cap", value: 105, max: 100, want: "##########"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bar(tt.value, tt.max); got != tt.want {
				t.Fatalf("bar(%f, %f) = %q, want %q", tt.value, tt.max, got, tt.want)
			}
		})
	}
}
