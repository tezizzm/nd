package model

import (
	"encoding/json"
	"testing"
)

func TestIssue_MarshalJSON_AllBlockedBy(t *testing.T) {
	i := Issue{
		ID:           "PROJ-x",
		Title:        "x",
		BlockedBy:    []string{"PROJ-b", "PROJ-a"},
		WasBlockedBy: []string{"PROJ-a", "PROJ-c"}, // PROJ-a overlaps -> deduped
	}

	data, err := json.Marshal(i)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got map[string]json.RawMessage
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Back-compat: the existing CamelCase edge fields are still emitted.
	for _, k := range []string{"BlockedBy", "WasBlockedBy", "AllBlockedBy"} {
		if _, ok := got[k]; !ok {
			t.Errorf("expected key %q in JSON, missing", k)
		}
	}

	var all []string
	if err := json.Unmarshal(got["AllBlockedBy"], &all); err != nil {
		t.Fatalf("AllBlockedBy not a string array: %v", err)
	}
	want := []string{"PROJ-a", "PROJ-b", "PROJ-c"} // deduplicated + sorted
	if len(all) != len(want) {
		t.Fatalf("AllBlockedBy = %v, want %v", all, want)
	}
	for idx := range want {
		if all[idx] != want[idx] {
			t.Errorf("AllBlockedBy = %v, want %v", all, want)
			break
		}
	}
}

func TestIssue_MarshalJSON_NoBlockers(t *testing.T) {
	data, err := json.Marshal(Issue{ID: "PROJ-y", Title: "y"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got struct {
		AllBlockedBy []string `json:"AllBlockedBy"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.AllBlockedBy) != 0 {
		t.Errorf("expected empty AllBlockedBy for an unblocked issue, got %v", got.AllBlockedBy)
	}
}

// The JSON-only AllBlockedBy field must not break unmarshaling JSON back into an
// Issue (e.g. archive round-trips); the unknown key is simply ignored.
func TestIssue_MarshalJSON_RoundTripsIntoIssue(t *testing.T) {
	orig := Issue{ID: "PROJ-z", Title: "z", BlockedBy: []string{"PROJ-a"}}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back Issue
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("round-trip unmarshal failed: %v", err)
	}
	if back.ID != orig.ID || len(back.BlockedBy) != 1 || back.BlockedBy[0] != "PROJ-a" {
		t.Errorf("round-trip lost data: %+v", back)
	}
}
