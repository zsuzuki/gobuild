package main

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestInterpolateLiteral(t *testing.T) {
	Convey("GIVEN: A test dictionary", t, func() {
		dict := newDictionary()
		for _, v := range []string{"", "literal", "foobarbaz$"} {
			Convey(fmt.Sprintf("WHEN: Interpolating \"%s\"", v), func() {
				actual, err := Interpolate(v, dict)
				Convey(fmt.Sprintf("THEN: Should be \"%s\"", v), func() {
					So(err, ShouldBeNil)
					So(actual, ShouldEqual, v)
				})
			})
		}
	})
}

func TestInterpolateExpansion(t *testing.T) {
	Convey("GIVEN: A Test dictionary", t, func() {
		dict := newDictionary()
		type testCase struct {
			input    string
			expected string
		}
		Convey("AND GIVEN: Test cases", func() {
			cases := []testCase{
				{input: "${foo}", expected: "foo-value"},
				{input: "${baz}", expected: "baz-value, bar-value, foo-value"},
				{input: "${foo}$$${bar}", expected: "foo-value$bar-value, foo-value"},
				{input: "${mokeke}moke", expected: "moke"}}
			for _, v := range cases {
				Convey(fmt.Sprintf("WHEN: Interpolating \"%s\"", v.input), func() {
					actual, err := Interpolate(v.input, dict)
					Convey(fmt.Sprintf("THEN: Should be \"%s\"", v.expected), func() {
						So(err, ShouldBeNil)
						So(actual, ShouldEqual, v.expected)
					})
				})
			}
		})
	})
}

func TestInterpolateError(t *testing.T) {
	dict := newDictionary()

	Convey("Errors", t, func() {
		Convey("Unmatched {}", func() {
			_, err := Interpolate("${foo", dict)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "{foo")
		})
		Convey("Passthrough unrecognized", func() {
			actual, err := Interpolate("$foo", dict)
			So(err, ShouldBeNil)
			So(actual, ShouldEqual, "$foo")
		})
		Convey("Exceeding recursion limit", func() {
			_, err := Interpolate("${rec}", dict)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "recursion limit exceeded")
		})
	})
}

func TestStrictInterpolateLiteral(t *testing.T) {
	dict := newDictionary()

	Convey("Test with literals", t, func() {
		Convey("\"\" should be \"\"", func() {
			actual, err := StrictInterpolate("", dict)
			So(err, ShouldBeNil)
			So(actual, ShouldEqual, "")
		})
		Convey("\"literal\" should be \"literal\"", func() {
			actual, err := StrictInterpolate("literal", dict)
			So(err, ShouldBeNil)
			So(actual, ShouldEqual, "literal")
		})
		Convey("\"foobarbaz$\" should be \"foobarbaz$\"", func() {
			actual, err := StrictInterpolate("foobarbaz$", dict)
			So(err, ShouldBeNil)
			So(actual, ShouldEqual, "foobarbaz$")
		})
	})
}

func TestStrictInterpolateExpansion(t *testing.T) {
	dict := newDictionary()
	Convey("Test expansions", t, func() {
		Convey("Single level expansion", func() {
			actual, err := StrictInterpolate("${foo}", dict)
			So(err, ShouldBeNil)
			So(actual, ShouldEqual, "foo-value")
		})
		Convey("Nested expansion", func() {
			actual, err := StrictInterpolate("${baz}", dict)
			So(err, ShouldBeNil)
			So(actual, ShouldEqual, "baz-value, bar-value, foo-value")
		})
		Convey("Double $$ in string", func() {
			actual, err := StrictInterpolate("${foo}$$${bar}", dict)
			So(err, ShouldBeNil)
			So(actual, ShouldEqual, "foo-value$bar-value, foo-value")
		})
	})
}
func TestStrictInterpolateErrors(t *testing.T) {
	dict := newDictionary()
	Convey("Errors", t, func() {
		Convey("Unmatched {}", func() {
			_, err := StrictInterpolate("${foo", dict)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "{foo")
		})
		Convey("Unrecognized as error", func() {
			_, err := StrictInterpolate("$foo", dict)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "invalid `$` sequence \"foo\" found.")
		})
		Convey("Non existants", func() {
			_, err := StrictInterpolate("${mokeke}moke", dict)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "unknown reference ${mokeke} found.")
		})
		Convey("Exceeding recursion limit", func() {
			_, err := StrictInterpolate("${rec}", dict)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "recursion limit exceeded")
		})
	})
}

func newDictionary() map[string]string {
	return map[string]string{
		"foo": "foo-value",
		"bar": "bar-value, ${foo}",
		"baz": "baz-value, ${bar}",
		"rec": "do${rec}"}
}
