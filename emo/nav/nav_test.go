package nav

import "testing"

func TestNavigateAndBack(t *testing.T) {
	// Reset state
	Reset("/")

	if Current().Path != "/" {
		t.Fatalf("initial path = %q, want /", Current().Path)
	}

	Navigate("/profile")
	if Current().Path != "/profile" {
		t.Fatalf("after navigate: path = %q, want /profile", Current().Path)
	}

	Navigate("/settings")
	if Current().Path != "/settings" {
		t.Fatalf("after navigate: path = %q, want /settings", Current().Path)
	}

	if !CanGoBack() {
		t.Fatal("CanGoBack should be true")
	}

	Back()
	if Current().Path != "/profile" {
		t.Fatalf("after back: path = %q, want /profile", Current().Path)
	}

	Back()
	if Current().Path != "/" {
		t.Fatalf("after back: path = %q, want /", Current().Path)
	}

	if Back() {
		t.Fatal("Back on root should return false")
	}
}

func TestReplace(t *testing.T) {
	Reset("/")
	Navigate("/profile")
	Replace("/settings")
	if Current().Path != "/settings" {
		t.Fatalf("after replace: path = %q, want /settings", Current().Path)
	}
	// Should only have 2 routes in stack (/, /settings)
	if len(Stack()) != 2 {
		t.Fatalf("stack length = %d, want 2", len(Stack()))
	}
}

func TestDynamicRouteParams(t *testing.T) {
	RegisterRoute("user/[id]", "user/:id")
	Navigate("/user/42")
	if Current().Screen != "user/[id]" {
		t.Fatalf("screen = %q, want user/[id]", Current().Screen)
	}
	if Param("id") != "42" {
		t.Fatalf("param id = %q, want 42", Param("id"))
	}
}

func TestOnNavigate(t *testing.T) {
	Reset("/")
	called := false
	unsub := OnNavigate(func(r Route) {
		called = true
	})
	Navigate("/test")
	if !called {
		t.Fatal("listener not called")
	}
	unsub()

	called = false
	Navigate("/test2")
	if called {
		t.Fatal("listener called after unsubscribe")
	}
}

func TestParseURL(t *testing.T) {
	// This tests the linking package's Parse, but we'll test nav's matchRoute
	RegisterRoute("post/[id]", "post/:id")
	Navigate("/post/123")
	if Current().Screen != "post/[id]" {
		t.Fatalf("screen = %q, want post/[id]", Current().Screen)
	}
	if Param("id") != "123" {
		t.Fatalf("param id = %q, want 123", Param("id"))
	}
}
