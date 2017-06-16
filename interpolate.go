// Interpolate `${foo}` style strings.

package main

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
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

// Interpolates supplied `s` using `dict`.
// If `s` contains unknown reference, treat it as an empty string ("").
func Interpolate(s string, dict map[string]string) (string, error) {
	return interpolate("", s, dict, 0, false)
}

// Interplates supplied `s` using `dict`.
// If `s` contains unknown reference, treat it as an error.
func InterpolateStrict (s string, dict map [string]string) (string, error) {
	return interpolate("", s, dict, 0, true)
}

func interpolate(accum string, rest string, dict map[string]string, limit int, strict bool) (string, error) {
	if recursionLimit <= limit {
		return "", errors.New("Recursion limit exceeded.")
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
				k := r[sz:sz+keyIdx]
				v, ok := dict[k]
				if strict {
					if ! ok {
						return accum, errors.New (fmt.Sprintf ("Unknown reference ${%s} found.", k))
					}
				}
				return interpolate(accum, v+r[sz+keyIdx+1:], dict, limit+1, strict)
			}
			return accum, errors.New(fmt.Sprintf("Unmatched `{` found after \"%s\".", r))
		case '$':
			return interpolate(accum+"$", r[sz:], dict, limit+1, strict)
		default:
			// Invalid $... sequence
			return accum, errors.New(fmt.Sprintf("Invalid `$` sequence \"%s\" found.", r))
		}
	} else {
		// No `$` in `s`
		return accum + rest, nil
	}
}
