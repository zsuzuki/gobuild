package main

import (
	"testing"

	"fmt"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBuildInfo_OptionPrefix(t *testing.T) {
	Convey("GIVEN: A `BuildInfo` with empty variable definitioins", t, func() {
		info := BuildInfo{}
		Convey("WHEN: Call `OptionPrefix`", func() {
			actual := info.OptionPrefix()
			Convey("THEN: Should return \"-\"", func() {
				So(actual, ShouldEqual, "-")
			})
		})
		Convey("WHEN: Defines `option_prefix = \"/\"`", func() {
			info.variables = map[string]string{"option_prefix": "/"}
			Convey("AND WHEN: Call `OptionPrefix`", func() {
				actual := info.OptionPrefix()
				Convey("THEN: Should return \"/\"", func() {
					So(actual, ShouldEqual, "/")
				})
			})
		})
	})
}

func TestBuildInfo_AddInclude(t *testing.T) {
	Convey("GIVEN: An empty `BuildInfo`", t, func() {
		info := BuildInfo{}
		for _, opfx := range []string{"/", "--"} {
			Convey(fmt.Sprintf("WHEN: option_prefix = \"%s\"", opfx), func() {
				info.variables = map[string]string{"option_prefix": opfx}
				Convey("AND WHEN: Call `AddInclude (\"/usr/local\")`", func() {
					info.AddInclude("/usr/local")
					Convey(fmt.Sprintf("THEN: info.includes should be [\"%sI/usr/local\"]", opfx), func() {
						So(len(info.includes), ShouldEqual, 1)
						So(info.includes, ShouldContain, fmt.Sprintf("%sI/usr/local", opfx))
					})
					Convey("AND WHEN: Call `AddInclude (\"/usr/foo bar\")`", func() {
						info.AddInclude("/usr/foo bar")
						Convey(fmt.Sprintf("THEN: info includes should be [\"%[1]sI/usr/local\", \"\\\"%[1]sI/usr/foo bar\\\"\"]", opfx), func() {
							So(len(info.includes), ShouldEqual, 2)
							So(info.includes[0], ShouldEqual, fmt.Sprintf("%sI/usr/local", opfx))
							So(info.includes[1], ShouldEqual, fmt.Sprintf("\"%sI/usr/foo bar\"", opfx))
						})
					})
				})
			})
		}
	})
}

func TestBuildInfo_AddDefines(t *testing.T) {
	Convey("GIVEN: An empty `BuildInfo`", t, func() {
		info := BuildInfo{}
		for _, opfx := range []string{"/", "--"} {
			Convey(fmt.Sprintf("WHEN: option_prefix = \"%s\"", opfx), func() {
				info.variables = map[string]string{"option_prefix": opfx}
				Convey("AND WHEN: Call `AddDefine (\"FOO\")`", func() {
					info.AddDefines("FOO")
					expected := fmt.Sprintf("%sDFOO", opfx)
					Convey(fmt.Sprintf("THEN: info.defines should be [\"%s\"]", expected), func() {
						So(len(info.defines), ShouldEqual, 1)
						So(info.defines, ShouldContain, expected)
					})
					Convey("AND WHEN: Call `AddDefine (\"BAR=BAZ\")`", func() {
						info.AddDefines("BAR=BAZ")
						Convey(fmt.Sprintf("THEN: info.defines should be [\"%[1]sDFOO\", \"%[1]sDBAR=BAZ\"]", opfx), func() {
							So(len(info.defines), ShouldEqual, 2)
							So(info.defines[0], ShouldEqual, fmt.Sprintf("%sDFOO", opfx))
							So(info.defines[1], ShouldEqual, fmt.Sprintf("%sDBAR=BAZ", opfx))
						})
					})
				})
			})
		}
	})
}
