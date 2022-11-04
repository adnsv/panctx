package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// wildcards are fragment in text file conforming to the following syntax `$<pref:cont>$``

var reWildcard = regexp.MustCompile(`(?m)\$<((?:[^>\$])*)>\$`)

var ErrInvalidVariableSyntax = errors.New("unsupported syntax")

func parseWildcardContent(s string) (k, v string, err error) {
	ss := strings.SplitN(s, ":", 2)
	if len(ss) < 2 {
		return "", "", ErrInvalidVariableSyntax
	} else {
		return ss[0], ss[1], nil
	}
}

func ForAllWildcards(buf []byte, handler func(k, v string) error) error {
	for _, v := range reWildcard.FindAllIndex(buf, -1) {
		b, e := v[0], v[1]
		s := string(buf[b+2 : e-2])
		k, v, err := parseWildcardContent(s)
		if err == nil {
			err = handler(k, v)
		}
		if err != nil {
			sl := CalcSourceLocation(string(buf), b)
			return fmt.Errorf("[%s] invalid wildcard: %w", sl.String(), err)
		}
	}
	return nil
}

type replaceCallback = func(pre, cnt string) (string, error)

func ReplaceWildcards(buf []byte, handler replaceCallback) ([]byte, error) {
	var firstErr error
	out := reWildcard.ReplaceAllFunc(buf, func(v []byte) []byte {
		if firstErr != nil || len(v) < 4 {
			return v
		}

		s := string(v[2 : len(v)-2])
		pre, cnt, err := parseWildcardContent(s)
		if err != nil {
			firstErr = err
			return v
		}
		s, err = handler(pre, cnt)
		if err != nil {
			firstErr = err
			return v
		}

		return []byte(s)
	})
	return out, firstErr
}
