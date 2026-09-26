package common

import "testing"

func TestGetLimit(t *testing.T) {
	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{"默认", 0, 10},
		{"正常", 20, 20},
		{"过大", 200, 10},
		{"负数", -1, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := PageInfo{Limit: tt.limit}
			got := p.GetLimit()
			if got != tt.want {
				t.Errorf("GetLimit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPage(t *testing.T) {
	tests := []struct {
		name string
		page int
		want int
	}{
		{"默认0", 0, 1},
		{"正常", 3, 3},
		{"过大", 21, 1},
		{"负数", -2, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := PageInfo{Page: tt.page}
			got := p.GetPage()
			if got != tt.want {
				t.Errorf("GetPage(%d) = %d, want %d", tt.page, got, tt.want)
			}
		})
	}
}
