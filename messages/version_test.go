package messages

import "testing"

func TestAppVersionTuple(t *testing.T) {
	if AppVersion != "2.3.0" {
		t.Fatalf("AppVersion = %q", AppVersion)
	}
	if appVersionTuple != [3]uint32{2, 3, 0} {
		t.Fatalf("appVersionTuple = %v, want [2 3 0]", appVersionTuple)
	}
}
