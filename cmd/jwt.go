package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"foxkit/internal/jwt"
	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "jwt <token|->",
		Short: "Xem header, payload, hạn dùng, kiểm tra chữ ký HS*",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			secret, _ := c.Flags().GetString("secret")
			token := args[0]
			if token == "-" {
				b, err := readStdin(c, 1<<20, "1MB")
				if err != nil {
					return err
				}
				token = string(b)
			}
			d, err := jwt.Decode(token, time.Now())
			if err != nil {
				return jwtError(err)
			}
			verified := -1 // -1: không verify, 0: sai, 1: đúng
			if secret != "" {
				ok, err := jwt.VerifyHMAC(token, secret)
				if err != nil {
					return jwtError(err)
				}
				d.Verified = &ok
				verified = 0
				if ok {
					verified = 1
				}
			}
			perr := printer(c).Success(d, func(w io.Writer) error {
				pretty := func(v any, indent string) string {
					b, _ := json.MarshalIndent(v, "", indent) // map đã decode từ JSON nên luôn marshal được
					return string(b)
				}
				hb, _ := json.Marshal(d.Header)
				fmt.Fprintf(w, "Header   %s\n", hb)
				fmt.Fprintf(w, "Payload  %s\n", strings.ReplaceAll(pretty(d.Payload, "  "), "\n", "\n         "))
				for _, k := range []string{"exp", "iat", "nbf"} {
					if t, ok := d.Times[k]; ok {
						state := ""
						if k == "exp" && d.Expired != nil {
							state = "   ✓ còn hạn"
							if *d.Expired {
								state = "   ✗ đã hết hạn"
							}
						}
						fmt.Fprintf(w, "%-8s %s  (%s)%s\n", k, time.Unix(t.Unix, 0).Format("2006-01-02 15:04:05"), t.Relative, state)
					}
				}
				switch verified {
				case 1:
					fmt.Fprintln(w, "Chữ ký   ✓ hợp lệ với secret đã cho")
				case 0:
					fmt.Fprintln(w, "Chữ ký   ✗ KHÔNG khớp với secret đã cho")
				}
				for _, warn := range d.Warnings {
					fmt.Fprintf(w, "Cảnh báo %s\n", warn)
				}
				return nil
			})
			if perr != nil {
				return perr
			}
			if verified == 0 {
				return output.Fail("JWT_SIGNATURE_INVALID", "Chữ ký không khớp với secret", "Kiểm tra lại secret")
			}
			return nil
		},
	}
	c.Flags().String("secret", "", "Secret để kiểm tra chữ ký HS*")
	rootCmd.AddCommand(c)
}

func jwtError(err error) error {
	switch {
	case errors.Is(err, jwt.ErrUnsupported):
		return output.Fail("JWT_UNSUPPORTED", err.Error(), "Chỉ hỗ trợ verify HS256/HS384/HS512")
	default:
		return output.Fail("JWT_INVALID", err.Error(), "Token JWT có dạng xxx.yyy.zzz")
	}
}
