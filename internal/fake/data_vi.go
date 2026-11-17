package fake

// Họ, có trọng số (Nguyễn ~38%).
var viSurnames = []weighted{
	{"Nguyễn", 38}, {"Trần", 11}, {"Lê", 9}, {"Phạm", 7}, {"Hoàng", 5}, {"Huỳnh", 4}, {"Phan", 4}, {"Vũ", 4},
	{"Võ", 3}, {"Đặng", 3}, {"Bùi", 3}, {"Đỗ", 3}, {"Hồ", 2}, {"Ngô", 2}, {"Dương", 1}, {"Lý", 1},
}

// Tên đệm: "Văn" chỉ đi với tên nam, "Thị" chỉ đi với tên nữ.
var viMiddleMale = []string{"Văn", "Hữu", "Minh", "Quốc", "Đức", "Anh", "Công", "Thanh", "Gia", "Xuân"}
var viMiddleFemale = []string{"Thị", "Ngọc", "Thu", "Thanh", "Mai", "Kim", "Phương", "Bích", "Hồng", "Khánh"}

var viFirstMale = []string{"Hùng", "Dũng", "Nam", "Long", "Tuấn", "Minh", "Khoa", "Phúc", "Hải", "Bảo", "Thành", "Trung", "Quân", "Huy", "Sơn", "Kiên", "Đạt", "Việt", "Thắng", "Hiếu"}
var viFirstFemale = []string{"Lan", "Hương", "Linh", "Trang", "Hạnh", "Thảo", "Mai", "Ngân", "Nhung", "Yến", "Oanh", "Vy", "Chi", "Hà", "Phương", "Thủy", "Loan", "Quỳnh", "Diệp", "My"}

var viStreets = []string{"Lê Lợi", "Nguyễn Huệ", "Trần Hưng Đạo", "Hai Bà Trưng", "Lý Thường Kiệt", "Phan Chu Trinh", "Điện Biên Phủ", "Cách Mạng Tháng Tám", "Nguyễn Trãi", "Lê Duẩn", "Võ Văn Tần", "Pasteur", "Hoàng Diệu", "Trường Chinh", "Quang Trung"}

// Tỉnh/thành → quận/huyện (bảng nhỏ, đủ giống thật, không cố chính xác tuyệt đối).
var viPlaces = []struct {
	City      string
	Districts []string
}{
	{"Hà Nội", []string{"Ba Đình", "Hoàn Kiếm", "Đống Đa", "Cầu Giấy", "Thanh Xuân", "Hai Bà Trưng", "Tây Hồ"}},
	{"TP. Hồ Chí Minh", []string{"Quận 1", "Quận 3", "Quận 5", "Quận 7", "Bình Thạnh", "Phú Nhuận", "Tân Bình", "Thủ Đức"}},
	{"Đà Nẵng", []string{"Hải Châu", "Thanh Khê", "Sơn Trà", "Ngũ Hành Sơn", "Liên Chiểu"}},
	{"Hải Phòng", []string{"Hồng Bàng", "Lê Chân", "Ngô Quyền", "Kiến An"}},
	{"Cần Thơ", []string{"Ninh Kiều", "Bình Thủy", "Cái Răng", "Ô Môn"}},
	{"Huế", []string{"Phú Xuân", "Thuận Hóa", "Hương Thủy"}},
	{"Nha Trang", []string{"Lộc Thọ", "Vĩnh Hải", "Vạn Thắng"}},
	{"Đà Lạt", []string{"Phường 1", "Phường 3", "Phường 8", "Xuân Thọ"}},
	{"Biên Hòa", []string{"Tân Phong", "Trảng Dài", "Long Bình"}},
	{"Vinh", []string{"Hưng Bình", "Trường Thi", "Lê Lợi"}},
	{"Quy Nhơn", []string{"Nguyễn Văn Cừ", "Ngô Mây", "Trần Phú"}},
	{"Buôn Ma Thuột", []string{"Tân Lợi", "Thống Nhất", "Tân An"}},
	{"Vũng Tàu", []string{"Thắng Tam", "Phường 1", "Thắng Nhì"}},
	{"Hạ Long", []string{"Bạch Đằng", "Hồng Gai", "Bãi Cháy"}},
	{"Thái Nguyên", []string{"Trưng Vương", "Phan Đình Phùng", "Quang Trung"}},
	{"Nam Định", []string{"Trần Hưng Đạo", "Bà Triệu", "Cửa Nam"}},
	{"Rạch Giá", []string{"Vĩnh Thanh", "Vĩnh Bảo", "An Hòa"}},
	{"Pleiku", []string{"Hoa Lư", "Diên Hồng", "Yên Đỗ"}},
	{"Long Xuyên", []string{"Mỹ Long", "Mỹ Bình", "Mỹ Xuyên"}},
	{"Thanh Hóa", []string{"Điện Biên", "Ba Đình", "Đông Sơn"}},
}

var viCompanyKinds = []string{"Công ty TNHH", "Công ty Cổ phần", "Tập đoàn", "Công ty TNHH MTV"}
var viCompanyWords = []string{"Phương Nam", "Sao Việt", "Hoàn Cầu", "Thành Đạt", "An Phát", "Bình Minh", "Đại Dương", "Hòa Bình", "Kỹ Thuật Số", "Công Nghệ Xanh", "Thương Mại Dịch Vụ", "Xây Dựng", "Vạn Phúc", "Tân Thịnh", "Gia Huy"}

// Từ dùng cho field text (lorem ipsum tiếng Việt).
var viWords = []string{"hôm", "nay", "trời", "đẹp", "chúng", "tôi", "đi", "làm", "việc", "cùng", "nhau", "rất", "vui", "sản", "phẩm", "khách", "hàng", "đơn", "hàng", "giao", "nhanh", "chất", "lượng", "tốt", "giá", "cả", "hợp", "lý", "cảm", "ơn", "bạn", "đã", "ủng", "hộ", "dịch", "vụ", "tuyệt", "vời", "sẽ", "quay", "lại", "lần", "sau", "thành", "phố", "mùa", "xuân", "gia", "đình", "công", "ty"}
