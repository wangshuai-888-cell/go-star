package collect_api

import "testing"

func TestOpenCollectAllowed(t *testing.T) {
	tests := []struct {
		name        string
		confExists  bool
		openCollect bool
		want        bool
	}{
		{"无配置", false, true, false},
		{"有配置未公开", true, false, false},
		{"有配置已公开", true, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := openCollectAllowed(tt.confExists, tt.openCollect)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
