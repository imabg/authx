package validate

import (
	netmail "net/mail"
	"strings"
)

// LooksLikeEmail reports whether s is a normalized, plausible email address.
func LooksLikeEmail(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}
	addr, err := netmail.ParseAddress(email)
	if err != nil {
		return false
	}
	if !strings.EqualFold(addr.Address, email) {
		return false
	}
	at := strings.LastIndex(addr.Address, "@")
	if at <= 0 || at == len(addr.Address)-1 {
		return false
	}
	local := addr.Address[:at]
	domain := addr.Address[at+1:]
	if local == "" || strings.ContainsAny(local, " \t\r\n") {
		return false
	}
	return strings.Contains(domain, ".") && !strings.ContainsAny(domain, " \t\r\n")
}
