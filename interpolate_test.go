package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestInterpolateLiteral(t *testing.T) {
	dict := newDictionary()

	Convey("Test with literals", t, func() {
		Convey("\"\" should be \"\"", func() {
			actual, err := Interpolate("", dict)
			So(err, ShouldBeNil)
			So(actual, ShouldEqual, "")
		})
		Convey("\"literal\" should be \"literal\"", func() {
			actual, err := Interpolate("literal", dict)
			So(err, ShouldBeNil)
			So(actual, ShouldEqual, "literal")
		})
		Convey("\"foobarbaz$\" should be \"foobarbaz$\"", func() {
			actual, err := Interpolate("foobarbaz$", dict)
			So(err, ShouldBeNil)
			So(actual, ShouldEqual, "foobarbaz$")
		})
	})
}

func TestInterpolateExpansion(t *testing.T) {
	dict := newDictionary()

	Convey("Test expansions", t, func() {
		Convey("Single level expansion", func() {
			actual, err := Interpolate("${foo}", dict)
			So(err, ShouldBeNil)
			So(actual, ShouldEqual, "foo-value")
		})
		Convey("Nested expansion", func() {
			actual, err := Interpolate("${baz}", dict)
			So(err, ShouldBeNil)
			So(actual, ShouldEqual, "baz-value, bar-value, foo-value")
		})
		Convey("Double $$ in string", func() {
			actual, err := Interpolate("${foo}$$${bar}", dict)
			So(err, ShouldBeNil)
			So(actual, ShouldEqual, "foo-value$bar-value, foo-value")
		})
		Convey("Non existants", func() {
			actual, err := Interpolate("${mokeke}moke", dict)
			So(err, ShouldBeNil)
			So(actual, ShouldEqual, "moke")
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
