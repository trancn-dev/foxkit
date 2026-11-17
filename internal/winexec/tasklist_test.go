package winexec

import "testing"

func TestParseTasklist(t *testing.T) {
	tests := []struct {
		name, in string
		want     []Proc
	}{
		{"comma", "\"node.exe\",\"1234\",\"Console\",\"1\",\"412,345 K\"\n", []Proc{{1234, "node.exe", 412345 * 1024}}},
		{"dot", "\"a.exe\",\"7\",\"Services\",\"0\",\"1.024 K\"\n", []Proc{{7, "a.exe", 1024 * 1024}}},
		{"no tasks", "INFO: No tasks are running\n", []Proc{}},
		{"empty", "", []Proc{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseTasklist(tc.in)
			if err != nil || len(got) != len(tc.want) {
				t.Fatalf("got %v err %v", got, err)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v want %v", got[i], tc.want[i])
				}
			}
		})
	}
}
