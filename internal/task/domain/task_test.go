package domain

import (
	"testing"
	"time"

	"secunda-task-management-service/internal/shared"
)

func TestNewTaskStoresOptionalFields(t *testing.T) {
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	task, err := NewTask(
		7,
		"  write tests  ",
		shared.Some("details"),
		2,
		shared.Some(int64(3)),
		now,
	)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}

	if task.Title != "write tests" {
		t.Errorf("Title = %q, want trimmed title", task.Title)
	}
	if description, ok := task.Description.TryUnwrap(); !ok || description != "details" {
		t.Errorf("Description = %v, want Some(details)", task.Description)
	}
	if assigneeID, ok := task.AssigneeID.TryUnwrap(); !ok || assigneeID != 3 {
		t.Errorf("AssigneeID = %v, want Some(3)", task.AssigneeID)
	}
	if task.ClosedAt.IsSome() {
		t.Errorf("ClosedAt = %v, want None", task.ClosedAt)
	}
}

func TestApplyUpdateMaintainsClosedAt(t *testing.T) {
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	task, err := NewTask(7, "task", shared.None[string](), 2, shared.None[int64](), now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}

	changes, err := task.ApplyUpdate(
		TaskUpdate{Status: shared.Some(StatusDone)},
		now.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("ApplyUpdate(done) error = %v", err)
	}
	if _, ok := changes["closed_at"]; !ok {
		t.Fatal("ApplyUpdate(done) changes missing closed_at")
	}
	closedAt, ok := task.ClosedAt.TryUnwrap()
	if !ok || !closedAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("ClosedAt = %v, want %v", task.ClosedAt, now.Add(time.Hour))
	}

	changes, err = task.ApplyUpdate(
		TaskUpdate{Status: shared.Some(StatusInProgress)},
		now.Add(2*time.Hour),
	)
	if err != nil {
		t.Fatalf("ApplyUpdate(in_progress) error = %v", err)
	}
	if _, ok := changes["closed_at"]; !ok {
		t.Fatal("ApplyUpdate(in_progress) changes missing closed_at")
	}
	if task.ClosedAt.IsSome() {
		t.Fatalf("ClosedAt = %v, want None", task.ClosedAt)
	}
}

func TestApplyUpdateNoneIsNoChange(t *testing.T) {
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	task, err := NewTask(7, "task", shared.None[string](), 2, shared.None[int64](), now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}

	if _, err := task.ApplyUpdate(TaskUpdate{}, now); err == nil {
		t.Fatal("ApplyUpdate() error = nil, want no changes error")
	}
}
