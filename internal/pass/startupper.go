package pass

// GenerateStartUpper sinh mật khẩu đủ 4 loại ký tự (charset "all") và bắt đầu bằng chữ in hoa.
// Chọn bằng cách sinh lại cho tới khi ký tự đầu là chữ hoa (≈ 3 lần/mật khẩu) để vẫn phân bố đều, không thiên lệch.
func GenerateStartUpper(length int, noAmbiguous bool) (string, error) {
	for try := 0; try < 10000; try++ {
		s, err := Generate(Options{Length: length, Charset: "all", NoAmbiguous: noAmbiguous})
		if err != nil {
			return "", err
		}
		if s[0] >= 'A' && s[0] <= 'Z' {
			return s, nil
		}
	}
	return Generate(Options{Length: length, Charset: "all", NoAmbiguous: noAmbiguous}) // không bao giờ tới đây trong thực tế
}
