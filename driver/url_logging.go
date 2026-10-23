package driver

import "net/url"

func urlForLog(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "<invalid URL>"
	}
	u.RawQuery = ""
	u.ForceQuery = false
	u.Fragment = ""
	u.User = nil
	return u.String()
}

func requestError(err error) error {
	if requestErr, ok := err.(*url.Error); ok {
		return requestErr.Err
	}
	return err
}
