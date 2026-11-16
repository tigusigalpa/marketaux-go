package marketaux

import (
	"net/url"
	"strings"
)

const redacted = "***"

// RedactURL returns raw with the api_token query parameter masked. It is used
// for error messages and debug logging so the API token never leaks. Any
// input that is not a parseable URL is returned with the token substring
// masked as well.
func RedactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return maskTokenParam(raw)
	}
	q := u.Query()
	if _, ok := q["api_token"]; !ok {
		return u.String()
	}
	q.Set("api_token", redacted)
	u.RawQuery = q.Encode()
	return u.String()
}

// redactValues encodes q and masks api_token. The result is always sorted and
// safe to place into errors and logs.
func redactValues(q url.Values) string {
	if _, ok := q["api_token"]; ok {
		q2 := make(url.Values, len(q))
		for k, vs := range q {
			q2[k] = append([]string(nil), vs...)
		}
		q2.Set("api_token", redacted)
		return q2.Encode()
	}
	return q.Encode()
}

// maskTokenParam best-effort masks api_token=... inside an unparseable raw URL.
func maskTokenParam(s string) string {
	const key = "api_token="
	var b strings.Builder
	for {
		i := strings.Index(s, key)
		if i < 0 {
			return b.String() + s
		}
		b.WriteString(s[:i+len(key)])
		s = s[i+len(key):]
		end := 0
		for end < len(s) && s[end] != '&' && s[end] != '#' && s[end] != ' ' {
			end++
		}
		b.WriteString(redacted)
		s = s[end:]
	}
}
