package core_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

type capabilityFixture struct {
	CapabilityID string   `json:"capability_id"`
	Phase        string   `json:"phase"`
	Scope        string   `json:"scope"`
	PythonSource []string `json:"python_source"`
	GoTargets    struct {
		Tests    []string `json:"tests"`
		Examples []string `json:"examples"`
		Impl     []string `json:"impl"`
	} `json:"go_targets"`
	Status    string `json:"status"`
	Milestone string `json:"milestone"`
}

func TestCapabilityMatrixFixtures(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	fixtureDir := filepath.Join(root, "tests", "fixtures", "capability_matrix")
	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatalf("read capability_matrix fixtures failed: %v", err)
	}

	validStatus := map[string]bool{"PASS": true, "PARTIAL": true, "MISSING": true}
	seen := map[string]string{}
	loaded := 0

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(fixtureDir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read fixture %s failed: %v", entry.Name(), err)
		}
		var items []capabilityFixture
		if err := json.Unmarshal(raw, &items); err != nil {
			t.Fatalf("parse fixture %s failed: %v", entry.Name(), err)
		}
		if len(items) == 0 {
			t.Fatalf("fixture %s must not be empty", entry.Name())
		}

		for _, item := range items {
			loaded++
			if item.CapabilityID == "" {
				t.Fatalf("fixture %s contains empty capability_id", entry.Name())
			}
			if prev, ok := seen[item.CapabilityID]; ok {
				t.Fatalf("duplicate capability_id=%s in %s and %s", item.CapabilityID, prev, entry.Name())
			}
			seen[item.CapabilityID] = entry.Name()
			if item.Phase == "" || item.Scope == "" {
				t.Fatalf("capability %s missing phase/scope", item.CapabilityID)
			}
			if !validStatus[item.Status] {
				t.Fatalf("capability %s has invalid status=%s", item.CapabilityID, item.Status)
			}
			if len(item.PythonSource) == 0 {
				t.Fatalf("capability %s must include python_source", item.CapabilityID)
			}
			if item.Status != "PASS" && item.Milestone == "" {
				t.Fatalf("capability %s with status=%s must include milestone", item.CapabilityID, item.Status)
			}

			for _, rel := range append(append(item.GoTargets.Tests, item.GoTargets.Examples...), item.GoTargets.Impl...) {
				if rel == "" {
					continue
				}
				abs := filepath.Join(root, rel)
				if _, err := os.Stat(abs); err != nil {
					t.Fatalf("capability %s target not found: %s", item.CapabilityID, rel)
				}
			}
		}
	}

	if loaded == 0 {
		t.Fatalf("no capability fixtures loaded from %s", fixtureDir)
	}

	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) < 10 {
		t.Fatalf("expected at least 10 capability entries, got %d", len(ids))
	}
}
