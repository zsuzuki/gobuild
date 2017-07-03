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

const recursionLimit = 500

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
	return interpolate(dict, false, s)
}

// StrictInterpolate interpolates supplied `s` using `dict`.
// If `s` contains unknown reference, treat it as an error.
func StrictInterpolate(s string, dict map[string]string) (string, error) {
	return interpolate(dict, true, s)
}

func interpolate(dict map[string]string, strict bool, rest string) (string, error) {
	depth := 0
	result := ""
restart:
	for {
		if recursionLimit <= depth {
			return result, newError(ExceedRecursionLimit, rest)
		}
		if idx := strings.Index(rest, "$"); 0 <= idx {
			// `$` found
			result += rest[:idx]
			r := rest[idx+1:]
			if len(r) == 0 {
				// "...$"
				return result + "$", nil
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
							return result, newError(UnknownReference, rest)
						}
					}
					if ix := strings.Index(v, "${"); 0 <= ix {
						depth += 1
						result += v[:ix]
						rest = v[ix:] + r[sz+keyIdx+1:]
					} else {
						result += v
						rest = r[sz+keyIdx+1:]
					}
					goto restart
				}
				return result, &InterpolationError{Type: UnmatchedBrace, Arg: rest}
			case '$':
				result += "$"
				rest = r[sz:]
				goto restart
			default:
				if strict {
					// Invalid $... sequence
					return result, newError(InvalidDollarSequence, rest)
				}
				// Treat it as is...
				result += "$"
				rest = r
				goto restart
			}
		} else {
			// No `$` in `s`
			result += rest
			return result, nil
		}
	}
}

func newError(t ErrorType, arg string) *InterpolationError {
	return &InterpolationError{Type: t, Arg: arg}
}
