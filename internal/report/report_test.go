package report

import (
	"encoding/json"
	"errors"
	"testing"
)

func validFail() Finding {
	return Finding{
		Module: "M2", RuleID: "duration_mismatch", Status: Fail,
		ByteOffset: 0xB042, FieldPath: "moov/trak[0]/edts/elst[1].media_time",
		Values:  []Value{{Name: "stts_sum", Value: 100, Unit: "tick"}, {Name: "elst_sum", Value: 102, Unit: "tick"}},
		Delta:   &Delta{Value: 2, Unit: "tick"},
		Message: "three durations disagree",
	}
}

func TestFailRequiresAllFourEvidenceFields(t *testing.T) {
	if err := validFail().Validate(); err != nil {
		t.Fatalf("complete finding rejected: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*Finding)
	}{
		{"no field_path", func(f *Finding) { f.FieldPath = "" }},
		{"no values", func(f *Finding) { f.Values = nil }},
		{"no delta", func(f *Finding) { f.Delta = nil }},
	}
	for _, tt := range tests {
		f := validFail()
		tt.mutate(&f)
		err := f.Validate()
		if err == nil {
			t.Errorf("%s: expected error, got nil", tt.name)
			continue
		}
		if !errors.Is(err, ErrIncompleteEvidence) {
			t.Errorf("%s: expected ErrIncompleteEvidence, got %v", tt.name, err)
		}
	}
}

func TestGrayReasonMustBeClosedSet(t *testing.T) {
	f := Finding{Module: "M3", Status: Gray, GrayReason: GrayVFRDetected}
	if err := f.Validate(); err != nil {
		t.Fatalf("closed-set reason rejected: %v", err)
	}
	// parse_error 不在闭集内 —— gray 不得兜住解析错误（D11）
	f.GrayReason = "parse_error"
	if err := f.Validate(); err == nil {
		t.Fatal("expected rejection of non-closed-set gray reason")
	}
	empty := Finding{Module: "M4", Status: Gray}
	if err := empty.Validate(); err == nil {
		t.Fatal("expected rejection of gray without reason")
	}
}

func TestBuildSummaryExcludesGrayFromFail(t *testing.T) {
	fs := []Finding{
		{Module: "M1", Status: Pass},
		{Module: "M2", Status: Gray, GrayReason: GrayDurationUnknown},
		{Module: "M1", Status: Fail, ByteOffset: 1, FieldPath: "x", Values: []Value{{Name: "a", Value: 1, Unit: "tick"}}, Delta: &Delta{Value: 1, Unit: "tick"}},
	}
	s := BuildSummary(fs)
	if s.Fail != 1 || s.Gray != 1 || s.Pass != 1 {
		t.Fatalf("summary = %+v", s)
	}
	if s.ByModule["M1"] != 1 || len(s.ByModule) != 1 {
		t.Fatalf("by-module fail counts wrong: %+v", s.ByModule)
	}
}

func TestSortFindingsIsDeterministic(t *testing.T) {
	mk := func(off uint64, path string) Finding {
		return Finding{Module: "M1", RuleID: "r", Status: Fail, ByteOffset: off, FieldPath: path,
			Values: []Value{{Name: "a", Value: 1, Unit: "byte"}}, Delta: &Delta{Value: 1, Unit: "byte"}}
	}
	a := []Finding{mk(20, "b"), mk(10, "z"), mk(10, "a")}
	b := []Finding{mk(10, "a"), mk(20, "b"), mk(10, "z")}
	SortFindings(a)
	SortFindings(b)
	for i := range a {
		if a[i].Fingerprint() != b[i].Fingerprint() {
			t.Fatalf("sort not deterministic at %d: %s vs %s", i, a[i].Fingerprint(), b[i].Fingerprint())
		}
	}
}

func TestJSONAndMarkdownShareSameFindings(t *testing.T) {
	rep := Report{Tool: "mediatrace", Version: "test", File: "x.mp4", Modules: []string{"M2"},
		Findings: []Finding{validFail()}}
	if err := rep.ValidateAll(); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	var back Report
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if len(back.Findings) != len(rep.Findings) {
		t.Fatalf("round-trip lost findings")
	}
	for i := range rep.Findings {
		if back.Findings[i].Fingerprint() != rep.Findings[i].Fingerprint() {
			t.Fatalf("fingerprint drift at %d", i)
		}
	}
	md := rep.RenderMarkdown()
	if len(md) == 0 {
		t.Fatal("markdown render empty")
	}
	if rep.VerdictLine() != "VERDICT: FAIL (1 findings: M2=1, gray=0)" {
		t.Fatalf("verdict line = %q", rep.VerdictLine())
	}
	if rep.FirstLine() == "FIRST: none" {
		t.Fatal("first line should point at the fail")
	}
}

func TestValidateAllRejectsBadFinding(t *testing.T) {
	rep := Report{Findings: []Finding{{Module: "M2", RuleID: "x", Status: Fail}}}
	if err := rep.ValidateAll(); err == nil {
		t.Fatal("expected validation failure for fail without evidence")
	}
}
