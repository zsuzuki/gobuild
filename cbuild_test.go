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

func TestToBoolean_Truthy(t *testing.T) {
	generator := gopter.CombineGens(
		gen.OneConstOf("true", "t", "yes", "y", "on", "1"),
		gen.AlphaString(),
	)
	condition := func(s string) bool {
		return ToBoolean(s)
	}
	Convey(`Truthy values should evaluate to true`, t, func() {
		So(condition, convey.ShouldSucceedForAll,
			generator.FlatMap(genVariants, reflect.TypeOf("")).WithLabel("truthy"))
	})
}

func TestToBoolean_Falsy(t *testing.T) {
	generator := gopter.CombineGens(
		gen.OneConstOf("false", "f", "no", "n", "off", "0"),
		gen.AlphaString(),
	)
	condition := func(s string) bool {
		return !ToBoolean(s)
	}
	Convey(`Truthy values should evaluate to true`, t, func() {
		So(condition, convey.ShouldSucceedForAll,
			generator.FlatMap(genVariants, reflect.TypeOf("")).WithLabel("falsy"))
	})
}

// genVariants constructs a generator for testing `ToBoolean`.
func genVariants(arg interface{}) gopter.Gen {
	args := arg.([]interface{})
	s := args[0].(string)
	t := args[1].(string)
	return gen.OneConstOf(s, strings.ToUpper(s), strings.Title(s),
		fmt.Sprintf("%s %s", s, t),
		fmt.Sprintf("%s %s", strings.ToUpper(s), t),
		fmt.Sprintf("%s %s", strings.Title(s), t),
	)
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

func TestReplaceExtension(t *testing.T) {
	condition := func(base string, ext string, newExt string) bool {
		var o string
		if strings.HasPrefix(ext, ".") {
			o = base + ext
		} else {
			o = base + "." + ext
		}
		s := ReplaceExtension(o, newExt)
		return o == ReplaceExtension(s, ext)
	}
	genBase := gopter.CombineGens(
		gen.Identifier(),
		gen.OneConstOf("", ".")).FlatMap(
		func(arg interface{}) gopter.Gen {
			args := arg.([]interface{})
			a0 := args[0].(string)
			a1 := args[1].(string)
			return gen.OneConstOf(a0, a0+a1)
		},
		reflect.TypeOf(""),
	)
	genExt := gen.Identifier()
	Convey(`Replace extention twice should match the original`, t, func() {
		So(condition, convey.ShouldSucceedForAll,
			genBase.WithLabel("base"),
			genExt.WithLabel("old"),
			genExt.WithLabel("new"))
	})
}

// To replace `.txt` to `.bin`
func ExampleReplaceExtension() {
	const f = "foo.bar.txt"
	fmt.Println(ReplaceExtension(f, ".bin"))
	// Output: foo.bar.bin
}
