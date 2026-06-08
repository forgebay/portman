package ports

import (
	"testing"

	psnet "github.com/shirou/gopsutil/v4/net"
)

func conn(status string, pid int32, port uint32) psnet.ConnectionStat {
	return psnet.ConnectionStat{
		Status: status,
		Pid:    pid,
		Laddr:  psnet.Addr{IP: "0.0.0.0", Port: port},
	}
}

func TestFilterListening(t *testing.T) {
	in := []psnet.ConnectionStat{
		conn("LISTEN", 100, 3000),
		conn("LISTEN", 100, 3000), // IPv6 duplicate of the same proc+port
		conn("ESTABLISHED", 100, 51000),
		conn("LISTEN", 0, 5432),  // system socket, PID hidden -> dropped
		conn("LISTEN", 200, 8000),
	}

	got := filterListening(in)

	if len(got) != 2 {
		t.Fatalf("expected 2 listeners, got %d: %+v", len(got), got)
	}
	want := map[int]int32{3000: 100, 8000: 200}
	for _, l := range got {
		if pid, ok := want[l.Port]; !ok || pid != l.PID {
			t.Errorf("unexpected listener %+v", l)
		}
	}
}

// TestListSmoke ensures List runs against the real OS without erroring and
// returns results sorted by port. It does not assert specific ports since the
// machine state is unknown.
func TestListSmoke(t *testing.T) {
	got, err := List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].Port > got[i].Port {
			t.Fatalf("results not sorted by port: %d before %d", got[i-1].Port, got[i].Port)
		}
	}
}
