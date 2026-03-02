package engines

import "testing"

func TestEngineKindStringAndStorage(t *testing.T) {
	tests := []struct {
		kind      EngineKind
		wantStr   string
		wantStore bool
	}{
		{kind: EngineKindS3, wantStr: "s3", wantStore: true},
		{kind: EngineKindDocker, wantStr: "docker", wantStore: false},
		{kind: EngineKindDump, wantStr: "dump", wantStore: false},
		{kind: EngineKind("custom"), wantStr: "custom", wantStore: false},
	}

	for _, tc := range tests {
		if got := tc.kind.String(); got != tc.wantStr {
			t.Fatalf("kind %q: expected String()=%q, got %q", tc.kind, tc.wantStr, got)
		}
		if got := tc.kind.IsStorageEngine(); got != tc.wantStore {
			t.Fatalf("kind %q: expected IsStorageEngine()=%v, got %v", tc.kind, tc.wantStore, got)
		}
	}
}
