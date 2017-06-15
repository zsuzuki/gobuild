// Interpolate `${foo}` style strings.

package main

import (
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

// Interpolates supplied `s` using.
func Interpolate(s string, dict map[string]string) (string, error) {
	return interpolate("", s, dict, 0)
}

func interpolate(accum string, rest string, dict map[string]string, limit int) (string, error) {
	if recursionLimit <= limit {
		return "", MyError{"Recursion limit exceeded."}
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
				v := dict[r[sz:sz+keyIdx]]
				return interpolate(accum, v+r[sz+keyIdx+1:], dict, limit+1)
			}
			return accum, MyError{fmt.Sprintf("Unmatched `{` found after \"%s\".", r)}
		case '$':
			return interpolate(accum+"$", r[sz:], dict, limit+1)
		default:
			// Invalid $... sequence
			return accum, MyError{fmt.Sprintf("Invalid `$` sequence \"%s\" found.", r)}
		}
	} else {
		// No `$` in `s`
		return accum + rest, nil
	}
}
