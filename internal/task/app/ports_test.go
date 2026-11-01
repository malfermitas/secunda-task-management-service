package app

import (
	"testing"

	"secunda-task-management-service/internal/shared"
	taskdomain "secunda-task-management-service/internal/task/domain"
)

func TestListInputFilterKey(t *testing.T) {
	tests := []struct {
		name string
		in   ListInput
		want string
	}{
		{
			name: "no filters",
			in:   ListInput{Limit: 20, Offset: 0},
			want: "status:all:assignee:all:limit:20:offset:0",
		},
		{
			name: "status and assignee",
			in: ListInput{
				Status:     shared.Some(taskdomain.StatusDone),
				AssigneeID: shared.Some(int64(42)),
				Limit:      50,
				Offset:     100,
			},
			want: "status:done:assignee:42:limit:50:offset:100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.FilterKey(); got != tt.want {
				t.Fatalf("FilterKey() = %q, want %q", got, tt.want)
			}
		})
	}
}
