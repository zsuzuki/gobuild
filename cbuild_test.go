package main

import (
	"testing"
	"fmt"

	. "github.com/smartystreets/goconvey/convey"
)

func TestToBooleanTrue(t *testing.T) {
	Convey("Test `ToBoolean` return true case", t, func() {
		trueCases := []string{
			"true", "True", "TRUE", "T",
			"yes", "Yes", "YES", "Y",
			"on", "On", "ON",
			"1"}
		for _, v := range trueCases {
			Convey(fmt.Sprintf("\"%s\" should be `true`", v), func() {
				actual := ToBoolean(v)
				So(actual, ShouldBeTrue)
			})
			addJunk := v + " ABCDEFG"
			Convey(fmt.Sprintf("\"%s\" should be `true`", addJunk), func() {
				actual := ToBoolean(v)
				So(actual, ShouldBeTrue)
			})
		}

	})
}

func TestToBooleanFalse(t *testing.T) {
	Convey("Test `ToBoolean` returns `false` case", t, func() {
		falseCases := []string{
			"false", "False", "FALSE", "F",
			"no", "No", "NO", "N",
			"off", "Off", "OFF",
			"0"}
		for _, v := range falseCases {
			Convey(fmt.Sprintf("\"%s\" should be `false`", v), func() {
				actual := ToBoolean(v)
				So(actual, ShouldBeFalse)
			})
			addJunk := v + " ABCDEFG"
			Convey(fmt.Sprintf("\"%s\" should be `false`", addJunk), func() {
				actual := ToBoolean(v)
				So(actual, ShouldBeFalse)
			})
		}
	})
}

func TestToBooleanUndefined(t *testing.T) {
	Convey("Test `ToBoolean` return `false` to unidentified string", t, func() {
		unknownCases := []string{
			"mokeke", "truthy", "offset", "2"}
		for _, v := range unknownCases {
			Convey(fmt.Sprintf("\"%s\" should be `false`", v), func() {
				actual := ToBoolean(v)
				So(actual, ShouldBeFalse)
			})
			addJunk := v + " ABCDEFG"
			Convey(fmt.Sprintf("\"%s\" should be `false`", addJunk), func() {
				actual := ToBoolean(v)
				So(actual, ShouldBeFalse)
			})
		}
	})
}
