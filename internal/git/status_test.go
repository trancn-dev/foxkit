package git

import "testing"

func TestParseStatusV2(t *testing.T) {
	tests := []struct {
		name, in string
		want     Status
	}{
		{"sạch", "# branch.oid abc\n# branch.head main\n# branch.upstream origin/main\n# branch.ab +0 -0\n", Status{Branch: "main", Upstream: "origin/main"}},
		{"ahead 2 behind 1", "# branch.head dev\n# branch.upstream origin/dev\n# branch.ab +2 -1\n", Status{Branch: "dev", Upstream: "origin/dev", Ahead: 2, Behind: 1}},
		{"staged+modified cùng file", "# branch.head main\n1 MM N... 100644 100644 100644 aaa bbb a.txt\n", Status{Branch: "main", Staged: 1, Modified: 1}},
		{"chỉ modified và chỉ staged", "# branch.head main\n1 .M N... 1 1 1 a b x\n1 A. N... 0 1 1 a b y\n", Status{Branch: "main", Staged: 1, Modified: 1}},
		{"untracked", "# branch.head main\n? new.txt\n? other.txt\n", Status{Branch: "main", Untracked: 2}},
		{"conflict", "# branch.head main\nu UU N... 1 1 1 1 a b c m.txt\n", Status{Branch: "main", Conflicts: 1}},
		{"rename", "# branch.head main\n2 R. N... 1 1 1 a b R100 new.txt\told.txt\n", Status{Branch: "main", Staged: 1}},
		{"detached", "# branch.head (detached)\n", Status{Branch: "(detached)"}},
		{"chưa có commit", "# branch.oid (initial)\n# branch.head main\n", Status{Branch: "main"}},
		{"CRLF", "# branch.head main\r\n? a\r\n", Status{Branch: "main", Untracked: 1}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseStatusV2([]byte(tc.in))
			if err != nil || got != tc.want {
				t.Fatalf("got %+v err %v, want %+v", got, err, tc.want)
			}
		})
	}
	if _, err := ParseStatusV2([]byte("# branch.ab +x -y\n")); err == nil {
		t.Error("branch.ab hỏng phải lỗi")
	}
}
