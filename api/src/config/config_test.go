package config

import "testing"

func TestConfigAuthRedirectURL(t *testing.T) {
	cfg := Config{
		DefaultRedirectURL:  "http://localhost:3000/mypage",
		AllowedRedirectURLs: []string{"http://localhost:3000/mypage", "http://localhost:3100", "https://app.example.com/dashboard"},
	}

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "empty redirect uses default redirect url",
			raw:  "",
			want: "http://localhost:3000/mypage",
		},
		{
			name: "allowed local relative redirect is returned",
			raw:  "/mypage?from=test",
			want: "http://localhost:3000/mypage?from=test",
		},
		{
			name: "unlisted relative redirect falls back to default redirect url",
			raw:  "/projects/1",
			want: "http://localhost:3000/mypage",
		},
		{
			name: "allowed external application redirect is returned",
			raw:  "http://localhost:3100/dashboard",
			want: "http://localhost:3100/dashboard",
		},
		{
			name: "allowed absolute redirect is returned",
			raw:  "https://app.example.com/dashboard/tasks",
			want: "https://app.example.com/dashboard/tasks",
		},
		{
			name: "untrusted host falls back to default redirect url",
			raw:  "https://evil.example.com/dashboard",
			want: "http://localhost:3000/mypage",
		},
		{
			name: "scheme relative redirect falls back to default redirect url",
			raw:  "//evil.example.com/dashboard",
			want: "http://localhost:3000/mypage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.AuthRedirectURL(tt.raw)
			if got != tt.want {
				t.Fatalf("AuthRedirectURL(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestConfigCORSOrigins(t *testing.T) {
	cfg := Config{
		FrontendURL:         "http://localhost:3000",
		DefaultRedirectURL:  "http://localhost:3000/mypage",
		AllowedRedirectURLs: []string{"http://localhost:3000/mypage", "http://localhost:3100/dashboard", "https://app.example.com/dashboard"},
	}

	got := cfg.CORSOrigins()
	want := []string{"http://localhost:3000", "http://localhost:3100", "https://app.example.com"}
	if len(got) != len(want) {
		t.Fatalf("CORSOrigins() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("CORSOrigins() = %v, want %v", got, want)
		}
	}
}
