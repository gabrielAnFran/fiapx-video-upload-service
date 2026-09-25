package dto

import "testing"

func TestNewProblem(t *testing.T) {
	p := NewProblem(404, "Not Found", "video not found", "/videos/123")

	if p.Type != "about:blank" {
		t.Errorf("Type = %q, want %q", p.Type, "about:blank")
	}
	if p.Title != "Not Found" {
		t.Errorf("Title = %q, want %q", p.Title, "Not Found")
	}
	if p.Status != 404 {
		t.Errorf("Status = %d, want 404", p.Status)
	}
	if p.Detail != "video not found" {
		t.Errorf("Detail = %q, want %q", p.Detail, "video not found")
	}
	if p.Instance != "/videos/123" {
		t.Errorf("Instance = %q, want %q", p.Instance, "/videos/123")
	}
}
