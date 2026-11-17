package rename

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// renameFn tách ra để test mô phỏng lỗi giữa chừng (≈ mock trong PHPUnit).
var renameFn = os.Rename

var ErrConflict = errors.New("kế hoạch đổi tên có xung đột")

type step struct{ from, to string }

// Apply đổi tên 2 pha (tên tạm → tên đích) để xử lý hoán đổi a↔b và đổi chỉ khác hoa thường.
// Có lỗi giữa chừng thì hoàn tác mọi bước đã xong theo thứ tự ngược rồi trả lỗi.
func Apply(p Plan) (int, error) {
	if len(p.Conflicts) > 0 {
		return 0, ErrConflict
	}
	var done []step
	rollback := func() {
		for i := len(done) - 1; i >= 0; i-- {
			_ = os.Rename(done[i].to, done[i].from) // hoàn tác tốt nhất có thể; lỗi hoàn tác không còn cách xử lý nào hơn
		}
	}
	tmps := make([]string, len(p.Changes))
	for i, c := range p.Changes {
		var b [4]byte
		if _, err := rand.Read(b[:]); err != nil {
			rollback()
			return 0, err
		}
		tmps[i] = filepath.Join(c.Dir, fmt.Sprintf(".%s.foxkit-tmp-%s", c.From, hex.EncodeToString(b[:])))
		if err := renameFn(filepath.Join(c.Dir, c.From), tmps[i]); err != nil {
			rollback()
			return 0, err
		}
		done = append(done, step{filepath.Join(c.Dir, c.From), tmps[i]})
	}
	for i, c := range p.Changes {
		target := filepath.Join(c.Dir, c.To)
		if err := renameFn(tmps[i], target); err != nil {
			rollback()
			return 0, err
		}
		// bước pha 2 thay thế bước pha 1 của cùng file: hoàn tác phải đưa target → tên gốc
		for j := range done {
			if done[j].to == tmps[i] {
				done[j].to = target
			}
		}
	}
	return len(p.Changes), nil
}
