package main

import (
	"testing"

	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"path/filepath"
	"strings"
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
			expected := addPrefix(opfx+"I", "/usr/local", "/usr/foo bar")
			Convey(fmt.Sprintf("WHEN: option_prefix = \"%s\"", opfx), func() {
				info.variables = map[string]string{"option_prefix": opfx}
				Convey("AND WHEN: Call `AddInclude (\"/usr/local\")`", func() {
					info.AddInclude("/usr/local")
					Convey(fmt.Sprintf(`THEN: info.includes should be ["%s"]`, expected[0]), func() {
						So(info.includes, ShouldResemble, expected[:1])
					})
					Convey("AND WHEN: Call `AddInclude (\"/usr/foo bar\")`", func() {
						info.AddInclude("/usr/foo bar")
						Convey(fmt.Sprintf(`THEN: info includes should be ["%s", "%s"]`, expected[0], expected[1]), func() {
							So(info.includes, ShouldResemble, expected[:2])
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
			inputs := []string{"FOO", "BAR=BAZ", "FOO-BAR=BAZ", "FOO-BAR=BAZ-HOGE"}
			expected := addPrefix(opfx+"D", "FOO", "BAR=BAZ", "FOO_BAR=BAZ", "FOO_BAR=BAZ-HOGE")
			Convey(fmt.Sprintf(`WHEN: option_prefix = "%s"`, opfx), func() {
				info.variables = map[string]string{"option_prefix": opfx}
				for i := range inputs {
					Convey(fmt.Sprintf(`AND WHEN: Call AddDefine (%s)`,
						strings.Join(inputs[:i+1], " ")), func() {
						for _, v := range inputs[:i+1] {
							info.AddDefines(v)
						}
						Convey(fmt.Sprintf(`THEN: info.defines should be %v`, expected[:i+1]), func() {
							So(info.defines, ShouldResemble, expected[:i+1])
						})
					})
				}
			})
		}
	})
}

func TestBuildInfo_MakeExecutablePath(t *testing.T) {
	Convey(`GIVEN: A BuildInfo with .outputdir = "/usr/local"`, t, func() {
		info := BuildInfo{variables: map[string]string{}, outputdir: "/usr/local"}
		Convey(`WHEN: Call with "TEST"`, func() {
			actual := filepath.ToSlash(info.MakeExecutablePath("TEST"))
			Convey(`THEN: Should return "/usr/local/TEST"`, func() {
				So(actual, ShouldEqual, "/usr/local/TEST")
			})
		})
		Convey(`WHEN: Set ".THE-SUFFIX" as ${execute_suffix}`, func() {
			info.variables["execute_suffix"] = ".THE-SUFFIX"
			Convey(`AND WHEN: Call with "TEST"`, func() {
				actual := filepath.ToSlash(info.MakeExecutablePath("TEST"))
				Convey(`THEN: Should return "/usr/local/TEST.THE-SUFFIX"`, func() {
					So(actual, ShouldEqual, "/usr/local/TEST.THE-SUFFIX")
				})
			})
		})
	})
}

func addPrefix(pfx string, args ...string) []string {
	result := make([]string, 0, len(args))
	for _, v := range args {
		if strings.ContainsAny(v, " \t") {
			result = append(result, fmt.Sprintf("\"%s%s\"", pfx, v))
			continue
		}
		result = append(result, pfx+v)
	}
	return result
}
