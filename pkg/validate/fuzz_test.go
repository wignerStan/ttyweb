package validate

import "testing"

func FuzzSessionName(f *testing.F) {
	seeds := []string{
		"my-session", "session_123", "session.test", "",
		"a", "valid-name-123_test.txt", "../../../etc/passwd",
		"session name with spaces", "name/with/slashes",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, name string) {
		_ = SessionName(name)
	})
}

func FuzzPaneID(f *testing.F) {
	seeds := []string{
		"main:0.0", "session-1.%1", "",
		"../../../etc/passwd", "pane with spaces",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, id string) {
		_ = PaneID(id)
	})
}

func FuzzAPIURL(f *testing.F) {
	seeds := []string{
		"http://localhost:8080", "https://api.example.com/v1",
		"", "javascript:alert(1)", "not-a-url",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, url string) {
		_ = APIURL(url)
	})
}
