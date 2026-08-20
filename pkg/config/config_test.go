package config

import "testing"

func TestIsDevelopment(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{env: "dev", want: true},
		{env: "development", want: true},
		{env: "DEV", want: true},
		{env: "Development", want: true},
		{env: "  dev  ", want: true},
		{env: "production", want: false},
		{env: "prod", want: false},
		{env: "staging", want: false},
		{env: "test", want: false},
		{env: "", want: false},
	}
	for _, tt := range tests {
		if got := IsDevelopment(tt.env); got != tt.want {
			t.Errorf("IsDevelopment(%q) = %v, want %v", tt.env, got, tt.want)
		}
		cfg := ApplicationConfig{}
		cfg.App.ENV = tt.env
		if got := cfg.IsDevelopment(); got != tt.want {
			t.Errorf("ApplicationConfig{ENV:%q}.IsDevelopment() = %v, want %v", tt.env, got, tt.want)
		}
	}
}
