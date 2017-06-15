// Interpolate `${foo}` style strings.

package main

import (
	"strings"
)

/*
 * str : (s)*
 *     ;
 * s : (* empty *)
 *   | <literal>
 *   | ${<literal>}
 *   ;
 */

// Interpolates supplied `s` using.
func Interpolate(s string, dict map[string]string) (string, error) {
	if idx := strings.Index(s, "$"); 0 <= idx {
		// `$` found
		return "", nil
	} else {
		// No `$` in `s`
		return s, nil
	}
}
