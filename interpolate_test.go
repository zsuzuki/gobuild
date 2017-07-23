package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/convey"
	"github.com/leanovate/gopter/gen"
	. "github.com/smartystreets/goconvey/convey"
)

type testCase struct {
	input    string
	expected string
}

func TestInterpolateLiteral(t *testing.T) {
	Convey("GIVEN: A empty dictionary", t, func() {
		dict := make(map[string]string)
		g := gen.AnyString().
			SuchThat(
				func(arg interface{}) bool {
					s := arg.(string)
					return !strings.ContainsAny(s, `${`)
				}).
			FlatMap(
				func(arg interface{}) gopter.Gen {
					s := arg.(string)
					return gen.OneConstOf(s, s+"$")
				},
				reflect.TypeOf(""))

		Convey(`WHEN: Apply property tests`, func() {
			condition := func(s string) bool {
				actual, err := Interpolate(s, dict)
				return err == nil && actual == s
			}
			Convey(`THEN: Should success for all`, func() {
				So(condition, convey.ShouldSucceedForAll, g)
			})
		})
		Convey(`WHEN: Apply property tests (strict)`, func() {
			condition := func(s string) bool {
				actual, err := StrictInterpolate(s, dict)

				return err == nil && actual == s
			}
			Convey(`THEN: Should success for all`, func() {
				So(condition, convey.ShouldSucceedForAll, g)
			})
		})
	})
}

func TestInterpolateExpansion(t *testing.T) {
	Convey("GIVEN: A Test dictionary", t, func() {
		dict := newDictionary()
		Convey("AND GIVEN: Test cases", func() {
			for _, v := range []testCase{
				{input: "${foo}", expected: "foo-value"},
				{input: "${baz}", expected: "baz-value, bar-value, foo-value"},
				{input: "${foo}$$${bar}", expected: "foo-value$bar-value, foo-value"}} {
				Convey(fmt.Sprintf("WHEN: Interpolating \"%s\"", v.input), func() {
					actual, err := Interpolate(v.input, dict)
					Convey(fmt.Sprintf("THEN: Should be \"%s\"", v.expected), func() {
						So(err, ShouldBeNil)
						So(actual, ShouldEqual, v.expected)
					})
				})
				Convey(fmt.Sprintf("WHEN: Interpolating \"%s\" (strict mode)", v.input), func() {
					actual, err := StrictInterpolate(v.input, dict)
					Convey(fmt.Sprintf("THEN: Should be \"%s\"", v.expected), func() {
						So(err, ShouldBeNil)
						So(actual, ShouldEqual, v.expected)
					})
				})
			}
		})
		Convey("WHEN Interpolating \"${mokeke}moke\"", func() {
			actual, err := Interpolate("${mokeke}moke", dict)
			Convey("THEN: Should be \"${mokeke}moke\"", func() {
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, "moke")
			})
		})
		Convey("WHEN Interpolating \"${mokeke}moke\" (strict-mode)", func() {
			_, err := StrictInterpolate("${mokeke}moke", dict)
			Convey("THEN: Should cause an error", func() {
				So(err, ShouldNotBeNil)
				Convey("AND THEN: Should be UnknownReference error", func() {
					e, ok := err.(*InterpolationError)
					So(ok, ShouldBeTrue)
					So(e.Type, ShouldEqual, UnknownReference)
				})
			})
		})
	})
}

func TestInterpolateError(t *testing.T) {
	Convey("GIVEN: A Test dictionary", t, func() {
		dict := newDictionary()
		Convey("WHEN: Interpolte \"${foo\" (Unmatched {})", func() {
			_, err := Interpolate("${foo", dict)
			Convey("THEN: Should cause error", func() {
				So(err, ShouldNotBeNil)
				Convey("AND THEN: Should have UnmatchedBrace error type", func() {
					e, ok := err.(*InterpolationError)
					So(ok, ShouldBeTrue)
					So(e.Type, ShouldEqual, UnmatchedBrace)
				})
			})
		})
		Convey("WHEN: Interpolate \"$foo\" (Passthrough unrecognized)", func() {
			actual, err := Interpolate("$foo", dict)
			Convey("THEN: Should success", func() {
				So(err, ShouldBeNil)
				So(actual, ShouldEqual, "$foo")
			})
		})
		Convey("WHEN: Interpolate \"${rec}\" (Exceeding recursion limit)", func() {
			actual, err := Interpolate("${rec}", dict)
			Convey("THEN: Should cause error", func() {
				So(err, ShouldNotBeNil)
				Convey("AND THEN: Should have ExceedRecursionLimit error type", func() {
					e, ok := err.(*InterpolationError)
					So(ok, ShouldBeTrue)
					So(e.Type, ShouldEqual, ExceedRecursionLimit)
					Printf("actual: \"%s\"", actual)
				})
			})
		})
	})
}

func TestStrictInterpolateErrors(t *testing.T) {
	Convey("GIVEN: A new dictionary", t, func() {
		dict := newDictionary()
		type errTestCase struct {
			input string
			err   ErrorType
		}
		for _, tc := range []errTestCase{
			{"${foo", UnmatchedBrace},
			{"$foo", InvalidDollarSequence},
			{"${mokeke}moke", UnknownReference},
			{"${rec}", ExceedRecursionLimit}} {
			Convey(fmt.Sprintf("WHEN: Interpolating \"%s\"", tc.input), func() {
				_, err := StrictInterpolate(tc.input, dict)
				Convey("THEN: Should cause an error", func() {
					So(err, ShouldNotBeNil)
					Convey(fmt.Sprintf("AND THEN: Should have %s error type", tc.err.String()), func() {
						e, ok := err.(*InterpolationError)
						So(ok, ShouldBeTrue)
						So(e.Type, ShouldEqual, tc.err)
					})
				})
			})
		}
	})
}

func newDictionary() map[string]string {
	return map[string]string{
		"foo": "foo-value",
		"bar": "bar-value, ${foo}",
		"baz": "baz-value, ${bar}",
		"rec": "do${rec}"}
}
