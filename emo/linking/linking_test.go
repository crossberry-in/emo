package linking

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		url     string
		scheme  string
		host    string
		path    string
		query   string
	}{
		{"myapp://profile/42", "myapp", "profile", "/42", ""},
		{"myapp://profile/42?ref=notif", "myapp", "profile", "/42", "ref=notif"},
		{"https://example.com/page", "https", "example.com", "/page", ""},
		{"mailto:test@example.com", "mailto", "test@example.com", "", ""},
	}

	for _, tt := range tests {
		p := Parse(tt.url)
		if p.Scheme != tt.scheme {
			t.Errorf("Parse(%q).Scheme = %q, want %q", tt.url, p.Scheme, tt.scheme)
		}
		if p.Host != tt.host {
			t.Errorf("Parse(%q).Host = %q, want %q", tt.url, p.Host, tt.host)
		}
		if p.Path != tt.path {
			t.Errorf("Parse(%q).Path = %q, want %q", tt.url, p.Path, tt.path)
		}
		if p.Query != tt.query {
			t.Errorf("Parse(%q).Query = %q, want %q", tt.url, p.Query, tt.query)
		}
	}
}

func TestCreateURL(t *testing.T) {
	SetScheme("myapp")
	url := CreateURL("/profile/42")
	if url != "myapp:///profile/42" {
		t.Fatalf("CreateURL = %q, want myapp:///profile/42", url)
	}
}

func TestSetScheme(t *testing.T) {
	SetScheme("custom")
	if Scheme() != "custom" {
		t.Fatalf("Scheme = %q, want custom", Scheme())
	}
	// Reset for other tests
	SetScheme("emo")
}

func TestHandleIncomingURL(t *testing.T) {
	called := false
	unsub := AddListener(func(p ParsedURL) {
		called = true
		if p.Scheme != "myapp" {
			t.Errorf("scheme = %q, want myapp", p.Scheme)
		}
	})
	defer unsub()

	HandleIncomingURL("myapp://profile/42")

	if !called {
		t.Fatal("listener not called")
	}
}
