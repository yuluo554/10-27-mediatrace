package policy

import "testing"

func TestExitCodePriority(t *testing.T) {
	cases := []struct {
		fails, grays, want int
	}{
		{0, 0, ExitPass},
		{0, 1, ExitInconclusive},
		{1, 0, ExitFail},
		{1, 3, ExitFail}, // fail 优先于 gray
		{0, 9, ExitInconclusive},
	}
	for _, c := range cases {
		if got := ExitCodeFor(c.fails, c.grays); got != c.want {
			t.Errorf("ExitCodeFor(%d,%d)=%d want %d", c.fails, c.grays, got, c.want)
		}
	}
}

func TestDurationWithinTolerance(t *testing.T) {
	cases := []struct {
		a, b int64
		want bool
	}{
		{100, 100, true},
		{100, 101, true},  // 容差 1 tick
		{101, 100, true},  // 对称
		{100, 102, false}, // 超容差
		{0, 1, true},
	}
	for _, c := range cases {
		if got := DurationWithinTolerance(c.a, c.b); got != c.want {
			t.Errorf("DurationWithinTolerance(%d,%d)=%v want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestSquareIsExactForFixedPoint(t *testing.T) {
	// 65536 = 1.0 in 16.16; 平方用于正交性检查
	if got := Square(65536); got != 4294967296 {
		t.Fatalf("Square(65536)=%d want 4294967296", got)
	}
}
