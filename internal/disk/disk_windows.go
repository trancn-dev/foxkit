package disk

import (
	"sort"
	"time"

	"golang.org/x/sys/windows"
)

// probe lấy thông tin một ổ; gọi API trên ổ mạng mất kết nối có thể treo nên luôn chạy trong goroutine có timeout.
func probe(root string, warnPct float64) Drive {
	d := Drive{Root: root}
	p, err := windows.UTF16PtrFromString(root)
	if err != nil {
		d.Error = err.Error()
		return d
	}
	d.Type = TypeName(windows.GetDriveType(p))
	var label, fs [windows.MAX_PATH + 1]uint16
	if err := windows.GetVolumeInformation(p, &label[0], uint32(len(label)), nil, nil, nil, &fs[0], uint32(len(fs))); err != nil {
		if d.Type == "cdrom" {
			d.Error = "không có đĩa"
		} else {
			d.Error = "không truy cập được"
		}
		return d
	}
	d.Label, d.FS = windows.UTF16ToString(label[:]), windows.UTF16ToString(fs[:])
	var avail, total, free uint64
	if err := windows.GetDiskFreeSpaceEx(p, &avail, &total, &free); err != nil {
		d.Error = "không đọc được dung lượng"
		return d
	}
	d.TotalBytes, d.FreeBytes = total, avail
	d.Fill(warnPct)
	return d
}

// List liệt kê ổ đĩa. all=false ẩn ổ không sẵn sàng (CD trống, ổ mạng mất kết nối).
func List(all bool, warnPct float64) ([]Drive, error) {
	const failCriticalErrors = 0x0001 // không bật hộp thoại "Không có đĩa" của Windows
	old := windows.SetErrorMode(failCriticalErrors)
	defer windows.SetErrorMode(old)
	buf := make([]uint16, 254)
	n, err := windows.GetLogicalDriveStrings(uint32(len(buf)), &buf[0])
	if err != nil {
		return nil, err
	}
	roots := SplitMultiSZ(windows.UTF16ToString(buf[:n]))
	// UTF16ToString dừng ở NUL đầu tiên → ghép lại từ buffer thô
	roots = splitUTF16(buf[:n])
	results := make(chan Drive, len(roots))
	for _, r := range roots {
		go func(r string) {
			ch := make(chan Drive, 1)
			go func() { ch <- probe(r, warnPct) }()
			select {
			case d := <-ch:
				results <- d
			case <-time.After(3 * time.Second):
				results <- Drive{Root: r, Type: "unknown", Error: "timeout"}
			}
		}(r)
	}
	out := []Drive{}
	for range roots {
		if d := <-results; all || d.Error == "" {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Root < out[j].Root })
	return out, nil
}

// splitUTF16 tách buffer multi-sz UTF-16 theo NUL.
func splitUTF16(b []uint16) []string {
	var out []string
	start := 0
	for i, c := range b {
		if c == 0 {
			if i > start {
				out = append(out, windows.UTF16ToString(b[start:i]))
			}
			start = i + 1
		}
	}
	return out
}
