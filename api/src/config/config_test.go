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
		FrontendURL:        "http://localhost:3000",
		DefaultRedirectURL: "http://localhost:3000/mypage",
		CORSAllowedOrigins: []string{"http://localhost:3000", "https://api.example.com"},
	}

	got := cfg.CORSOrigins()
	want := []string{"http://localhost:3000", "https://api.example.com"}
	if len(got) != len(want) {
		t.Fatalf("CORSOrigins() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("CORSOrigins() = %v, want %v", got, want)
		}
	}
}

func TestSplitOriginsRejectsRedirectURLsWithPaths(t *testing.T) {
	got := splitOrigins("http://localhost:3000,https://app.example.com/dashboard,https://api.example.com")
	want := []string{"http://localhost:3000", "https://api.example.com"}
	if len(got) != len(want) {
		t.Fatalf("splitOrigins() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("splitOrigins() = %v, want %v", got, want)
		}
	}
}

// 空要素を除いて、信頼するプロキシ CIDR のリストを作ることを確認する。
func TestSplitStringsRemovesEmptyValues(t *testing.T) {
	got := splitStrings("10.0.0.0/8, ,192.0.2.10")
	want := []string{"10.0.0.0/8", "192.0.2.10"}
	if len(got) != len(want) {
		t.Fatalf("splitStrings() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("splitStrings() = %v, want %v", got, want)
		}
	}
}

// 空文字だけなら nil にして、Gin が forwarded header を信頼しないことを確認する。
func TestSplitStringsReturnsNilForEmptyValues(t *testing.T) {
	got := splitStrings(" , ")
	if got != nil {
		t.Fatalf("splitStrings() = %v, want nil", got)
	}
}

// 環境変数から、信頼するプロキシ CIDR を設定へ反映することを確認する。
func TestLoadReadsTrustedProxies(t *testing.T) {
	t.Setenv("GOOGLE_ID", "google-client-id")
	t.Setenv("GOOGLE_SECRET", "google-secret")
	t.Setenv("AUTH_SECRET", "01234567890123456789012345678901")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/app")
	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/8,192.0.2.10")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := []string{"10.0.0.0/8", "192.0.2.10"}
	if len(cfg.TrustedProxies) != len(want) {
		t.Fatalf("TrustedProxies = %v, want %v", cfg.TrustedProxies, want)
	}
	for i := range want {
		if cfg.TrustedProxies[i] != want[i] {
			t.Fatalf("TrustedProxies = %v, want %v", cfg.TrustedProxies, want)
		}
	}
}
