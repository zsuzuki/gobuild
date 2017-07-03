package main

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestToBooleanTrue(t *testing.T) {
	Convey("GIVEN: Test cases", t, func() {
		trueCases := []string{
			"true", "True", "TRUE", "T",
			"yes", "Yes", "YES", "Y",
			"on", "On", "ON",
			"1"}
		for _, v := range trueCases {
			Convey(fmt.Sprintf("WHEN: Evaluating \"%s\"", v), func() {
				actual := ToBoolean(v)
				Convey("THEN: Should be `true`", func() {
					So(actual, ShouldBeTrue)
				})
			})
		}
		for _, v := range trueCases {
			addJunk := v + " ABCDEFG"
			Convey(fmt.Sprintf("WHEN: Evaluationg \"%s\"", addJunk), func() {
				actual := ToBoolean(v)
				Convey("THEN: Should be `true`", func() {
					So(actual, ShouldBeTrue)
				})
			})
		}
	})
}

func TestToBooleanFalse(t *testing.T) {
	Convey("GIVEN: Test cases", t, func() {
		falseCases := []string{
			"false", "False", "FALSE", "F",
			"no", "No", "NO", "N",
			"off", "Off", "OFF",
			"0"}
		for _, v := range falseCases {
			Convey(fmt.Sprintf("WHEN: Evaluating \"%s\"", v), func() {
				actual := ToBoolean(v)
				Convey("THEN: Should be `false`", func() {
					So(actual, ShouldBeFalse)
				})
			})
		}
		for _, v := range falseCases {
			addJunk := v + " ABCDEFG"
			Convey(fmt.Sprintf("WHEN: Evaluating \"%s\"", addJunk), func() {
				actual := ToBoolean(v)
				Convey("THEN: Should be `false`", func() {
					So(actual, ShouldBeFalse)
				})
			})
		}
	})
}

func TestToBooleanUndefined(t *testing.T) {
	Convey("GIVEN: Test cases", t, func() {
		unknownCases := []string{
			"mokeke", "truthy", "offset", "2"}
		for _, v := range unknownCases {
			Convey(fmt.Sprintf("WHEN: Evaluating \"%s\"", v), func() {
				actual := ToBoolean(v)
				Convey("THEN: Should be `false`", func() {
					So(actual, ShouldBeFalse)
				})
			})
		}
		for _, v := range unknownCases {
			addJunk := v + " ABCDEFG"
			Convey(fmt.Sprintf("WHEN: Evaluating \"%s\"", addJunk), func() {
				actual := ToBoolean(v)
				Convey("THEN: Should be `false`", func() {
					So(actual, ShouldBeFalse)
				})
			})
		}
	})
}

func TestFixupCommandPath(t *testing.T) {
	Convey("GIVEN: Test cases", t, func() {
		cases := []struct {
			arg1st    string
			arg2nd    string
			expect1st string
			expect2nd string
		}{
			{"abc def ghi", "/usr/local", "/usr/local/abc def ghi", "/usr/local/abc"},
			{"$abc def ghi", "/usr/local", "/usr/local/$abc def ghi", "/usr/local/$abc"},
		}

		for _, c := range cases {
			Convey(fmt.Sprintf(`WHEN: Evaluating FixupCommandPath("%s", "%s")`, c.arg1st, c.arg2nd), func() {
				ea, eb := FixupCommandPath(c.arg1st, c.arg2nd)
				Convey(fmt.Sprintf("THEN: Should be (\"%s\", \"%s\")", c.expect1st, c.expect2nd), func() {
					So(ea, ShouldEqual, c.expect1st)
					So(eb, ShouldEqual, c.expect2nd)
				})
			})
		}
	})
}
