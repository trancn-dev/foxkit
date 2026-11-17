package textcase

import (
	"reflect"
	"testing"
)

func TestConvert(t *testing.T) {
	tests := []struct{ in, mode, want string }{
		{"userName", "snake", "user_name"},
		{"HTTPServer", "kebab", "http-server"},
		{"getHTTPResponse", "studly", "GetHttpResponse"},
		{"user_profile-id", "camel", "userProfileId"},
		{"Đơn hàng mới", "slug", "don-hang-moi"},
		{"Tiếng Việt có dấu", "snake", "tieng_viet_co_dau"},
		{"  nhiều   khoảng trắng ", "kebab", "nhieu-khoang-trang"},
		{"Ưu đãi 50%!", "slug", "uu-dai-50"},
		{"file2name", "snake", "file2name"},
		{"", "camel", ""},
		{"user profile ID", "constant", "USER_PROFILE_ID"},
		{"user profile ID", "title", "User Profile Id"},
		{"Đơn hàng", "upper", "ĐƠN HÀNG"},
		{"Đơn Hàng", "lower", "đơn hàng"},
		{"đơn hàng mới", "title", "Đơn Hàng Mới"},
		{"user profile ID", "pascal", "UserProfileId"},
	}
	for _, tc := range tests {
		got, err := Convert(tc.mode, tc.in)
		if err != nil || got != tc.want {
			t.Errorf("Convert(%s,%q)=%q,%v want %q", tc.mode, tc.in, got, err, tc.want)
		}
	}
	if _, err := Convert("zzz", "x"); err == nil {
		t.Error("cần lỗi mode lạ")
	}
}

func TestWords(t *testing.T) {
	got := Words("getHTTPResponse2x")
	want := []string{"get", "HTTP", "Response2x"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
	if got := Words(""); got == nil || len(got) != 0 {
		t.Fatalf("rỗng phải là slice rỗng, got %#v", got)
	}
}

func TestRemoveDiacritics(t *testing.T) {
	if got := RemoveDiacritics("Đơn hàng ĐẶNG Ưu"); got != "Don hang DANG Uu" {
		t.Fatalf("got %q", got)
	}
}
