// Interpolate `${foo}` style strings.

package main

import (
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"
)

/*
 * str : (s)*
 *     ;
 * s : (* empty *)
 *   | <literal>
 *   | ${<literal>}
 *   ;
 */

const recursionLimit = 100

// Interpolate interpolates supplied `s` using `dict`.
// If `s` contains unknown reference, treat it as an empty string ("").
func Interpolate(s string, dict map[string]string) (string, error) {
	return interpolate("", s, dict, 0, false)
}

// StrictInterpolate interplates supplied `s` using `dict`.
// If `s` contains unknown reference, treat it as an error.
func StrictInterpolate(s string, dict map[string]string) (string, error) {
	return interpolate("", s, dict, 0, true)
}

func interpolate(accum string, rest string, dict map[string]string, limit int, strict bool) (string, error) {
	if recursionLimit <= limit {
		return "", errors.New("recursion limit exceeded")
	}
	if idx := strings.Index(rest, "$"); 0 <= idx {
		// `$` found
		accum += rest[:idx]
		r := rest[idx+1:]
		if len(r) == 0 {
			// "...$"
			return accum + "$", nil
		}
		// 0 < len (r)
		ch, sz := utf8.DecodeRuneInString(r)
		switch ch {
		case '{':
			if keyIdx := strings.Index(r[sz:], "}"); 0 <= keyIdx {
				k := r[sz : sz+keyIdx]
				v, ok := dict[k]
				if strict {
					if !ok {
						return accum, errors.Errorf("unknown reference ${%s} found.", k)
					}
				}
				return interpolate(accum, v+r[sz+keyIdx+1:], dict, limit+1, strict)
			}
			return accum, errors.Errorf("unmatched `{` found after \"%s\".", r)
		case '$':
			return interpolate(accum+"$", r[sz:], dict, limit+1, strict)
		default:
			if strict {
				// Invalid $... sequence
				return accum, errors.Errorf("invalid `$` sequence \"%s\" found.", r)
			}
			// Treat it as is...
			return interpolate(accum+"$", r, dict, limit+1, false)
		}
	} else {
		// No `$` in `s`
		return accum + rest, nil
	}
}
