package rules

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestListRules_SortedByID(t *testing.T) {
	rules, err := ListRules()
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}

	if len(rules) == 0 {
		t.Fatal("expected at least one rule")
	}

	for i := 1; i < len(rules); i++ {
		if rules[i].ID < rules[i-1].ID {
			t.Errorf("rules not sorted: %s comes after %s", rules[i].ID, rules[i-1].ID)
		}
	}
}

func TestListRules_ContainsMDS001(t *testing.T) {
	rules, err := ListRules()
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}

	found := false
	for _, r := range rules {
		if r.ID == "MDS001" {
			found = true
			if r.Name != "line-length" {
				t.Errorf("MDS001 name = %q, want %q", r.Name, "line-length")
			}
			if r.Description == "" {
				t.Error("MDS001 description is empty")
			}
			break
		}
	}
	if !found {
		t.Error("MDS001 not found in rule list")
	}
}

func TestLookupRule_ByID(t *testing.T) {
	content, err := LookupRule("MDS001")
	if err != nil {
		t.Fatalf("LookupRule(MDS001): %v", err)
	}

	if !strings.Contains(content, "line-length") {
		t.Error("expected MDS001 content to contain 'line-length'")
	}
}

func TestLookupRule_ByName(t *testing.T) {
	content, err := LookupRule("line-length")
	if err != nil {
		t.Fatalf("LookupRule(line-length): %v", err)
	}

	if !strings.Contains(content, "MDS001") {
		t.Error("expected line-length content to contain 'MDS001'")
	}
}

func TestLookupRule_CaseInsensitiveID(t *testing.T) {
	content, err := LookupRule("mds001")
	if err != nil {
		t.Fatalf("LookupRule(mds001): %v", err)
	}

	if !strings.Contains(content, "MDS001") {
		t.Error("expected lowercase lookup to find MDS001")
	}
}

func TestLookupRule_Unknown(t *testing.T) {
	_, err := LookupRule("MDSXXX")
	if err == nil {
		t.Fatal("expected error for unknown rule")
	}
	if !strings.Contains(err.Error(), "unknown rule") {
		t.Errorf("error = %q, want it to contain 'unknown rule'", err.Error())
	}
}

func TestListRulesFromFS_SkipsBadFrontMatter(t *testing.T) {
	fsys := fstest.MapFS{
		"good/README.md": &fstest.MapFile{
			Data: []byte("---\nid: MDS999\nname: good-rule\nstatus: ready\ndescription: A good rule.\n---\n# MDS999\n"),
		},
		"bad/README.md": &fstest.MapFile{
			Data: []byte("no front matter here\n"),
		},
	}

	rules, err := listRulesFromFS(fsys)
	if err != nil {
		t.Fatalf("listRulesFromFS: %v", err)
	}

	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}

	if rules[0].ID != "MDS999" {
		t.Errorf("rule ID = %q, want MDS999", rules[0].ID)
	}
}

func TestLookupRuleFromFS_ByIDAndName(t *testing.T) {
	fsys := fstest.MapFS{
		"testrule/README.md": &fstest.MapFile{
			Data: []byte("---\nid: MDS999\nname: test-rule\nstatus: ready\ndescription: Test.\n---\n# Content\n"),
		},
	}

	content, err := lookupRuleFromFS(fsys, "MDS999")
	if err != nil {
		t.Fatalf("lookupRuleFromFS(MDS999): %v", err)
	}
	if !strings.Contains(content, "# Content") {
		t.Error("expected content to contain '# Content'")
	}

	content, err = lookupRuleFromFS(fsys, "test-rule")
	if err != nil {
		t.Fatalf("lookupRuleFromFS(test-rule): %v", err)
	}
	if !strings.Contains(content, "# Content") {
		t.Error("expected content to contain '# Content'")
	}
}

func TestLookupRuleFromFS_NotFound(t *testing.T) {
	fsys := fstest.MapFS{
		"testrule/README.md": &fstest.MapFile{
			Data: []byte("---\nid: MDS999\nname: test-rule\nstatus: ready\ndescription: Test.\n---\n# Content\n"),
		},
	}

	_, err := lookupRuleFromFS(fsys, "MDSXXX")
	if err == nil {
		t.Fatal("expected error for unknown rule")
	}
}

func TestListRulesFromFS_SkipsMissingStatus(t *testing.T) {
	fsys := fstest.MapFS{
		"nostatus/README.md": &fstest.MapFile{
			Data: []byte("---\nid: MDS998\nname: no-status\ndescription: Missing status.\n---\n# MDS998\n"),
		},
	}

	rules, err := listRulesFromFS(fsys)
	if err != nil {
		t.Fatalf("listRulesFromFS: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules, got %d", len(rules))
	}
}
