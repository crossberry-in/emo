package constants

import "testing"

func TestDefaults(t *testing.T) {
	if AppName() == "" {
		t.Fatal("AppName should not be empty")
	}
	if SdkVersion() == "" {
		t.Fatal("SdkVersion should not be empty")
	}
}

func TestSet(t *testing.T) {
	Set(Constants{
		AppName:    "test-app",
		AppVersion: "2.0.0",
	})
	if AppName() != "test-app" {
		t.Fatalf("AppName = %q, want test-app", AppName())
	}
	if AppVersion() != "2.0.0" {
		t.Fatalf("AppVersion = %q, want 2.0.0", AppVersion())
	}
}

func TestAll(t *testing.T) {
	Set(Constants{
		AppName:    "myapp",
		AppVersion: "1.0.0",
	})
	all := All()
	if all["appName"] != "myapp" {
		t.Fatalf("appName = %q, want myapp", all["appName"])
	}
}
