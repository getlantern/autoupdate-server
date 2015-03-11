package server

import (
	"testing"
)

func TestCompareVersions(t *testing.T) {
	if VersionCompare("v1.0.0", "v2.0.0") != Higher {
		t.Fatalf("Expecting value: Higher")
	}
	if VersionCompare("v1.0.0", "v1.0.0") != Equal {
		t.Fatalf("Expecting value: Equal")
	}
	if VersionCompare("v1.0.0", "v0.1.0") != Lower {
		t.Fatalf("Expecting value: Lower")
	}
	if VersionCompare("v1.1.99", "v1.1.999") != Higher {
		t.Fatalf("Expecting value: Higher")
	}
	if VersionCompare("v1.1.99.1", "v1.1.999") != Higher {
		t.Fatalf("Expecting value: Higher")
	}
	if VersionCompare("v1.1.0.0.0", "v1.1") != Equal {
		t.Fatalf("Expecting value: Equal")
	}
	if VersionCompare("v1.1", "v1.1.0.0") != Equal {
		t.Fatalf("Expecting value: Equal")
	}
	if VersionCompare("v1.1.", "v1.1.0.") != Equal {
		t.Fatalf("Expecting value: Equal")
	}
	if VersionCompare("v1.1.", "v1.1.0...") != Equal {
		t.Fatalf("Expecting value: Equal")
	}
	if VersionCompare("v1.0.0.1", "v1.0.0") != Lower {
		t.Fatalf("Expecting value: Lower")
	}
	if VersionCompare("v2.0.0.1.1", "v2.1.0.99") != Higher {
		t.Fatalf("Expecting value: Higher")
	}
	// We don't actually compare beta or alpha strings yet, this is just a test
	// for unexpected content.
	if VersionCompare("v2.0.0.1-beta1", "v2.0.0.1-beta2") != Higher {
		t.Fatalf("Expecting value: Higher")
	}
}
