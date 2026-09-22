package migrations

import (
	"slices"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/lib/pq"
)

func TestV2_10_0TasksMigration(t *testing.T) {
	db := testutil.NewDB(t, "migration_v2_10_0")

	// Start from roles and statuses as they were before tasks existed.
	if _, err := db.Exec(`UPDATE roles SET permissions = array_remove(array_remove(array_remove(array_remove(permissions,
		'tasks:read'), 'tasks:write'), 'tasks:delete'), 'tasks:manage')`); err != nil {
		t.Fatalf("resetting role permissions: %v", err)
	}

	// Running the migration twice verifies it is idempotent.
	for range 2 {
		if err := V2_10_0(db, nil, nil); err != nil {
			t.Fatalf("running migration: %v", err)
		}
	}

	var statuses int
	if err := db.Get(&statuses, `SELECT COUNT(*) FROM task_statuses`); err != nil {
		t.Fatal(err)
	}
	if statuses != 3 {
		t.Fatalf("want the 3 seeded statuses once, got %d", statuses)
	}

	want := map[string][]string{
		"Agent": {"tasks:read", "tasks:write", "tasks:delete"},
		"Admin": {"tasks:read", "tasks:write", "tasks:delete", "tasks:manage"},
	}
	for role, perms := range want {
		var got pq.StringArray
		if err := db.Get(&got, `SELECT permissions FROM roles WHERE name = $1`, role); err != nil {
			t.Fatalf("reading role %q: %v", role, err)
		}
		for _, p := range []string{"tasks:read", "tasks:write", "tasks:delete", "tasks:manage"} {
			count := 0
			for _, g := range got {
				if g == p {
					count++
				}
			}
			wantCount := 0
			if slices.Contains(perms, p) {
				wantCount = 1
			}
			if count != wantCount {
				t.Errorf("role %q: permission %q appears %d times, want %d", role, p, count, wantCount)
			}
		}
	}
}
