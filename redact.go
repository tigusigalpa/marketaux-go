package marketaux

import (
	"encoding/json"
	"net/url"
	"strings"
)

const redacted = "***"

// redactSecret also covers credentials echoed by servers and transports.
func (c *Client) redactSecret(s string) string {
	quoted, _ := json.Marshal(c.token)
	return strings.NewReplacer(
		url.QueryEscape(c.token), redacted,
		url.PathEscape(c.token), redacted,
		string(quoted[1:len(quoted)-1]), redacted,
		c.token, redacted,
	).Replace(s)
}

// diagnosticError preserves error identity while keeping its message safe.
type diagnosticError struct {
	err     error
	message string
}

func (e *diagnosticError) Error() string { return e.message }
func (e *diagnosticError) Unwrap() error { return e.err }

func (c *Client) safeError(err error) error {
	message := c.redactSecret(err.Error())
	if message == err.Error() {
		return err
	}
	return &diagnosticError{err: err, message: message}
}

// RedactURL returns raw with the api_token query parameter masked. It is used
// for error messages and debug logging so the API token never leaks. Any
// input that is not a parseable URL is replaced with "[invalid URL]".
func RedactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "[invalid URL]"
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		parts := strings.Split(u.RawQuery, "&")
		for i, part := range parts {
			key, _, _ := strings.Cut(part, "=")
			if decoded, decodeErr := url.QueryUnescape(key); decodeErr == nil && decoded == "api_token" {
				parts[i] = key + "=" + url.QueryEscape(redacted)
			} else {
				parts[i] = maskTokenParam(part)
			}
		}
		u.RawQuery = strings.Join(parts, "&")
		return u.String()
	}
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
