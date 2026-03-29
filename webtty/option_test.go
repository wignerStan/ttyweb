package webtty

import (
	"testing"
)

func TestWithFixedColumns(t *testing.T) {
	t.Parallel()
	opt := WithFixedColumns(120)
	wt := &WebTTY{}
	if err := opt(wt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wt.columns != 120 {
		t.Fatalf("expected columns 120, got %d", wt.columns)
	}
}

func TestWithFixedColumns_Zero(t *testing.T) {
	t.Parallel()
	opt := WithFixedColumns(0)
	wt := &WebTTY{}
	if err := opt(wt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wt.columns != 0 {
		t.Fatalf("expected columns 0, got %d", wt.columns)
	}
}

func TestWithFixedRows(t *testing.T) {
	t.Parallel()
	opt := WithFixedRows(40)
	wt := &WebTTY{}
	if err := opt(wt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wt.rows != 40 {
		t.Fatalf("expected rows 40, got %d", wt.rows)
	}
}

func TestWithFixedRows_Zero(t *testing.T) {
	t.Parallel()
	opt := WithFixedRows(0)
	wt := &WebTTY{}
	if err := opt(wt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wt.rows != 0 {
		t.Fatalf("expected rows 0, got %d", wt.rows)
	}
}

func TestWindowTitle(t *testing.T) {
	t.Parallel()
	title := []byte("My Terminal")
	opt := WithWindowTitle(title)
	wt := &WebTTY{}
	if err := opt(wt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(wt.windowTitle) != "My Terminal" {
		t.Fatalf("expected window title %q, got %q", "My Terminal", string(wt.windowTitle))
	}
}

func TestWindowTitle_Nil(t *testing.T) {
	t.Parallel()
	opt := WithWindowTitle(nil)
	wt := &WebTTY{}
	if err := opt(wt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wt.windowTitle != nil {
		t.Fatalf("expected nil window title, got %v", wt.windowTitle)
	}
}

func TestWindowTitle_Empty(t *testing.T) {
	t.Parallel()
	opt := WithWindowTitle([]byte{})
	wt := &WebTTY{}
	if err := opt(wt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wt.windowTitle) != 0 {
		t.Fatalf("expected empty window title, got %v", wt.windowTitle)
	}
}

func TestWithFixedColumnsAndRows_Combined(t *testing.T) {
	t.Parallel()
	wt := &WebTTY{}
	for _, opt := range []Option{
		WithFixedColumns(200),
		WithFixedRows(50),
	} {
		if err := opt(wt); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if wt.columns != 200 {
		t.Fatalf("expected columns 200, got %d", wt.columns)
	}
	if wt.rows != 50 {
		t.Fatalf("expected rows 50, got %d", wt.rows)
	}
}

func TestWithMasterPreferences_Unmarshallable(t *testing.T) {
	t.Parallel()
	opt := WithMasterPreferences(make(chan int))
	wt := &WebTTY{}
	err := opt(wt)
	if err == nil {
		t.Fatal("expected error for unmarshallable preferences")
	}
}
