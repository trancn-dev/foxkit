package wifi

import (
	"context"
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf16"

	"foxkit/internal/winexec"
)

type Profile struct {
	SSID     string `json:"ssid"`
	Auth     string `json:"auth"`
	Password string `json:"password,omitempty"`
	Hidden   bool   `json:"hidden"`
	Error    string `json:"error,omitempty"`
}

var ErrUnavailable = errors.New("không có Wi-Fi hoặc dịch vụ WLAN AutoConfig đang tắt")

// wlanProfile: encoding/xml khớp theo tên cục bộ nên bỏ qua namespace mặc định.
type wlanProfile struct {
	Name       string `xml:"name"`
	SSIDConfig struct {
		NonBroadcast bool `xml:"nonBroadcast"`
	} `xml:"SSIDConfig"`
	MSM struct {
		Security struct {
			AuthEncryption struct {
				Authentication string `xml:"authentication"`
			} `xml:"authEncryption"`
			SharedKey struct {
				KeyMaterial string `xml:"keyMaterial"`
				Protected   bool   `xml:"protected"`
			} `xml:"sharedKey"`
		} `xml:"security"`
	} `xml:"MSM"`
}

// ParseProfileXML đọc một file profile do `netsh wlan export profile` tạo ra.
func ParseProfileXML(b []byte) (Profile, error) {
	if len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE { // UTF-16 LE có BOM
		u := make([]uint16, (len(b)-2)/2)
		for i := range u {
			u[i] = uint16(b[2+2*i]) | uint16(b[3+2*i])<<8
		}
		s := string(utf16.Decode(u))
		if i := strings.Index(s, "?>"); strings.HasPrefix(s, "<?xml") && i > 0 {
			s = s[i+2:] // bỏ khai báo encoding="utf-16" vì giờ đã là UTF-8
		}
		b = []byte(s)
	}
	var w wlanProfile
	if err := xml.Unmarshal(b, &w); err != nil {
		return Profile{}, err
	}
	p := Profile{SSID: w.Name, Auth: w.MSM.Security.AuthEncryption.Authentication, Hidden: w.SSIDConfig.NonBroadcast}
	sk := w.MSM.Security.SharedKey
	switch {
	case sk.Protected:
		p.Error = "cần quyền admin để xem mật khẩu"
	default:
		p.Password = sk.KeyMaterial
	}
	if strings.EqualFold(p.Auth, "open") {
		p.Auth = "open"
	}
	return p, nil
}

// Export xuất profile ra thư mục tạm (chứa mật khẩu rõ nên luôn bị xóa), đọc rồi lọc theo ssid.
// ssid rỗng = mọi profile. Kết quả sắp theo SSID.
func Export(ctx context.Context, ssid string) ([]Profile, error) {
	dir, err := os.MkdirTemp("", "foxkit-wifi-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir) // bắt buộc xóa dù lỗi
	if _, err := winexec.Run(ctx, 20*time.Second, "netsh", "wlan", "export", "profile", "key=clear", "folder="+dir); err != nil {
		return nil, ErrUnavailable
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.xml"))
	if err != nil {
		return nil, err
	}
	out := []Profile{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		p, err := ParseProfileXML(b)
		if err != nil {
			continue
		}
		if ssid == "" || strings.EqualFold(p.SSID, ssid) {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].SSID) < strings.ToLower(out[j].SSID) })
	return out, nil
}
