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

// ErrorType means type of the error.
type ErrorType int

//go:generate stringer -type=ErrorType
const (
	// ExceedRecursionLimit means too many recursion occur while interpolation.
	ExceedRecursionLimit ErrorType = iota
	// UnmatchedBrace means unmatched brace found while interpolation.
	UnmatchedBrace
	// UnknownReference means undefined variable reference found while interpolation.
	// Note: Only used in strict-mode.
	UnknownReference
	// InvalidDollerSequence means there is an unknown `$` prefixed sequence in the argument.
	// Note: Only used in strict-mode
	InvalidDollarSequence
)

// InterpolationError holds the error information occurs inside interpolator.
type InterpolationError struct {
	// Type means what error occurred while interpolation.
	Type ErrorType
	// Arg holds the string cause this error.
	Arg string
}

func (e InterpolationError) Error() string {
	switch e.Type {
	case ExceedRecursionLimit:
		return fmt.Sprintf("recursion limit exceeded while interpolating \"%s\"", e.Arg)
	case UnmatchedBrace:
		return fmt.Sprintf("unmached brace found while interpolating \"%s\"", e.Arg)
	case UnknownReference:
		return fmt.Sprintf("unknown reference found while interpolating \"%s\"", e.Arg)
	case InvalidDollarSequence:
		return fmt.Sprintf("unknown '$' start sequence found while interpolating \"%s\"", e.Arg)
	default:
		return fmt.Sprintf("bad error type (%d)", int(e.Type))
	}
}

// Interpolate interpolates supplied `s` using `dict`.
// If `s` contains unknown reference, treat it as an empty string ("").
func Interpolate(s string, dict map[string]string) (string, error) {
	return interpolate("", s, dict, 0, false)
}

// StrictInterpolate interpolates supplied `s` using `dict`.
// If `s` contains unknown reference, treat it as an error.
func StrictInterpolate(s string, dict map[string]string) (string, error) {
	return interpolate("", s, dict, 0, true)
}

func interpolate(accum string, rest string, dict map[string]string, limit int, strict bool) (string, error) {
	if recursionLimit <= limit {
		return "", newError(ExceedRecursionLimit, rest)
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
						return accum, newError(UnknownReference, rest)
					}
				}
				return interpolate(accum, v+r[sz+keyIdx+1:], dict, limit+1, strict)
			}
			return accum, &InterpolationError{Type: UnmatchedBrace, Arg: rest}
		case '$':
			return interpolate(accum+"$", r[sz:], dict, limit+1, strict)
		default:
			if strict {
				// Invalid $... sequence
				return accum, newError(InvalidDollarSequence, rest)
			}
			// Treat it as is...
			return interpolate(accum+"$", r, dict, limit+1, false)
		}
	} else {
		// No `$` in `s`
		return accum + rest, nil
	}
}

func newError(t ErrorType, arg string) *InterpolationError {
	return &InterpolationError{Type: t, Arg: arg}
}
