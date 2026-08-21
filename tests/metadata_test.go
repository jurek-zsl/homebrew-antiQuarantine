package tests

import (
	"testing"
	"time"

	"antiQuarantine/internal/quarantine"
)

func TestParseQuarantineString(t *testing.T) {
	// Sample string: Flags=0081 (Quarantined + Network Download), Timestamp=65d8ab12 (1708706578), Agent=Safari, UUID=B8C27D56-5B81-4C3D-B9AC-06D76D38B1C8
	raw := "0081;65d8ab12;Safari;B8C27D56-5B81-4C3D-B9AC-06D76D38B1C8"
	meta, err := quarantine.ParseQuarantineString(raw)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if meta.FlagsHex != "0081" {
		t.Errorf("expected flags 0081, got %s", meta.FlagsHex)
	}
	if meta.Flags != 0x0081 {
		t.Errorf("expected flags uint 0x0081, got 0x%04X", meta.Flags)
	}
	if !meta.IsMalware {
		t.Errorf("expected IsMalware (quarantined) to be true")
	}
	if !meta.IsFromWeb {
		t.Errorf("expected IsFromWeb to be true")
	}
	if meta.IsApproved {
		t.Errorf("expected IsApproved to be false")
	}
	if meta.Agent != "Safari" {
		t.Errorf("expected Agent Safari, got %s", meta.Agent)
	}
	if meta.EventUUID != "B8C27D56-5B81-4C3D-B9AC-06D76D38B1C8" {
		t.Errorf("expected UUID match, got %s", meta.EventUUID)
	}

	expectedTime := time.Unix(0x65d8ab12, 0).UTC()
	if !meta.Timestamp.Equal(expectedTime) {
		t.Errorf("timestamp mismatch. Got %v, want %v", meta.Timestamp, expectedTime)
	}
}

func TestParseApprovedQuarantine(t *testing.T) {
	// 00c2 = 0x0040 (Approved) + 0x0080 (Web) + 0x0002 (Agent Assessed)
	raw := "00c2;65d8ab12;Google Chrome;12345678-1234-1234-1234-123456789abc"
	meta, err := quarantine.ParseQuarantineString(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !meta.IsApproved {
		t.Errorf("expected IsApproved to be true for flag 0x00c2")
	}
	if meta.Agent != "Google Chrome" {
		t.Errorf("expected Google Chrome, got %s", meta.Agent)
	}
}
