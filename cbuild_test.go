package main

import (
	"fmt"
	"testing"

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

func TestFixupCommandPath(t *testing.T) {
	Convey("Test `FixupCommandPath`", t, func() {
		cases := []struct {
			arg1st    string
			arg2nd    string
			expect1st string
			expect2nd string
		}{
			{"abc def ghi", "/usr/local",
				"/usr/local/abc def ghi", "/usr/local/abc"},
			{"$abc def ghi", "/usr/local",
				"/usr/local/$abc def ghi", "/usr/local/$abc"},
		}

		for _, c := range cases {
			Convey(fmt.Sprintf(`Test FixupCommandPath("%s", "%s") == ("%s", "%s")`,
				c.arg1st, c.arg2nd,
				c.expect1st, c.expect2nd), func() {

				ea, eb := FixupCommandPath(c.arg1st, c.arg2nd)
				So(ea, ShouldEqual, c.expect1st)
				So(eb, ShouldEqual, c.expect2nd)
			})
		}
	})
}
