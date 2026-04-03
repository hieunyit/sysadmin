package idempotency

import "net/http"

const Header = "Idempotency-Key"

func Get(r *http.Request) string {
	return r.Header.Get(Header)
}
