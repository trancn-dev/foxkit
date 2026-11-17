package port

import "testing"

func TestParseNetstat(t *testing.T) {
	tests := []struct {
		name, in string
		want     []Listener
	}{
		{"ipv4 listen", "  TCP    127.0.0.1:8000    0.0.0.0:0    LISTENING    12345", []Listener{{Port: 8000, Protocol: "TCP", Address: "127.0.0.1", PID: 12345}}},
		{"ipv6 listen", "  TCP    [::]:5173    [::]:0    LISTENING    9812", []Listener{{Port: 5173, Protocol: "TCP", Address: "::", PID: 9812}}},
		{"state translated", "  TCP    0.0.0.0:80    0.0.0.0:0    ABHÖREN    4", []Listener{{Port: 80, Protocol: "TCP", Address: "0.0.0.0", PID: 4}}},
		{"established ignored", "  TCP    127.0.0.1:52000    127.0.0.1:8000    ESTABLISHED    777", []Listener{}},
		{"header and junk", "  Proto  Local Address  Foreign Address  State  PID\n\nActive Connections\n", []Listener{}},
		{"udp ignored", "  UDP    0.0.0.0:5353    *:*    1234", []Listener{}},
		{"sorted both stacks", "  TCP    [::]:8000    [::]:0    LISTENING    1\n  TCP    0.0.0.0:80    0.0.0.0:0    LISTENING    2\n  TCP    0.0.0.0:8000    0.0.0.0:0    LISTENING    1", []Listener{
			{Port: 80, Protocol: "TCP", Address: "0.0.0.0", PID: 2},
			{Port: 8000, Protocol: "TCP", Address: "::", PID: 1},
			{Port: 8000, Protocol: "TCP", Address: "0.0.0.0", PID: 1},
		}},
		{"crlf", "  TCP    0.0.0.0:3306    0.0.0.0:0    LISTENING    2210\r\n", []Listener{{Port: 3306, Protocol: "TCP", Address: "0.0.0.0", PID: 2210}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseNetstat(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v want %v", got, tc.want)
				}
			}
		})
	}
}

func TestKillSystem(t *testing.T) {
	for _, pid := range []int{0, 4} {
		if err := Kill(pid); err != ErrSystemProcess {
			t.Errorf("Kill(%d) = %v", pid, err)
		}
	}
}
