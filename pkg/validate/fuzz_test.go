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
		"http://localhost:8080", "https://localhost/v1",
		"", "javascript:alert(1)", "://not-a-url",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, url string) {
		_ = APIURL(url)
	})
}

func TestFuzzCorpus_SeedValidation(t *testing.T) {
	t.Run("SessionName", func(t *testing.T) {
		assertValid(t, SessionName, "my-session", "session_123", "session.test", "a", "valid-name-123_test.txt")
		assertInvalid(t, SessionName, "", "../../../etc/passwd", "session name with spaces", "name/with/slashes")
	})
	t.Run("PaneID", func(t *testing.T) {
		assertValid(t, PaneID, "main:0.0", "session-1.%1")
		assertInvalid(t, PaneID, "", "../../../etc/passwd", "pane with spaces")
	})
	t.Run("APIURL", func(t *testing.T) {
		assertValid(t, APIURL, "http://localhost:8080", "https://localhost/v1")
		assertInvalid(t, APIURL, "", "javascript:alert(1)", "://not-a-url")
	})
}

func assertValid(t *testing.T, fn func(string) error, inputs ...string) {
	t.Helper()
	for _, v := range inputs {
		if err := fn(v); err != nil {
			t.Errorf("expected valid, got error: %v", v)
		}
	}
}

func assertInvalid(t *testing.T, fn func(string) error, inputs ...string) {
	t.Helper()
	for _, v := range inputs {
		if err := fn(v); err == nil {
			t.Errorf("expected invalid, got nil: %v", v)
		}
	}
}
