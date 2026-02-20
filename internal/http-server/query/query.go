package query

import (
	"fmt"
	"net/http"
	"strconv"
)

func Int(r *http.Request, key string) (val int, present bool, err error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return 0, false, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, true, fmt.Errorf("%s must be integer", key)
	}
	return n, true, nil
}

func IntAny(r *http.Request, keys ...string) (val int, present bool, err error) {
	for _, k := range keys {
		v, ok, e := Int(r, k)
		if e != nil {
			return 0, false, e
		}
		if ok {
			return v, true, nil
		}
	}
	return 0, false, nil
}
