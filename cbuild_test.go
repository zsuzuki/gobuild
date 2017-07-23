package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"path/filepath"

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
	Convey(`Falsy values should evaluate to false`, t, func() {
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
	Convey("GIVEN: A property", t, func() {
		condition := func(a0 string, a1 string) bool {
			args := strings.Split(a0, " ")
			ea, eb := FixupCommandPath(a0, a1)
			if filepath.ToSlash(filepath.Join(a1, args[0])) != filepath.ToSlash(eb) {
				return false
			}
			if filepath.ToSlash(filepath.Join(a1, a0)) != filepath.ToSlash(ea) {
				return false
			}
			return true
		}
		Convey("WHEN: Testing property", func() {
			Convey("THEN: Should succeed for all cases", func() {
				So(condition, convey.ShouldSucceedForAll,
					gen.SliceOf(gen.Identifier()).
						FlatMap(
							func(arg interface{}) gopter.Gen {
								args := arg.([]string)
								args2 := make([]string, 0, len(args))
								for _, t := range args {
									args2 = append(args2, t)
								}
								if 0 < len(args2) {
									args2[0] = "$" + args2[0]
								}
								return gen.OneConstOf(strings.Join(args, " "), strings.Join(args2, " "))
							},
							reflect.TypeOf("")).WithLabel("args"),
					genPath().WithLabel("path"))
			})
		})
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
		gen.OneConstOf("", "."),
	).FlatMap(
		func(arg interface{}) gopter.Gen {
			args := arg.([]interface{})
			a0 := args[0].(string)
			a1 := args[1].(string)
			return gen.OneConstOf(a0, a0+a1)
		},
		reflect.TypeOf(""),
	)
	Convey(`Replace extention twice should match the original`, t, func() {
		So(condition, convey.ShouldSucceedForAll,
			genBase.WithLabel("base"),
			genPathComponent(false).WithLabel("old"),
			genPathComponent(false).WithLabel("new"))
	})
}

func TestBasename(t *testing.T) {
	Convey(`Basename ("foo.bar", "bar") == "foo"`, t, func() {
		condition := func(dir string, path string, ext string) bool {
			fullpath := filepath.Join(dir, fmt.Sprintf("%s.%s", path, ext))
			//t.Logf("fullpath = %s", fullpath)
			p := Basename(fullpath, ext)
			return p == path
		}
		So(condition,
			convey.ShouldSucceedForAll,
			genPath().WithLabel("dir"),
			genPathComponent(true).WithLabel("stem"),
			genPathComponent(false).WithLabel("ext"))
	})
}

func genPath() gopter.Gen {
	pathGen := gen.SliceOf(genPathComponent(true)).Map(func(args []string) string {
		return filepath.ToSlash(filepath.Join(args...))
	})
	return gopter.CombineGens(genVolume(), pathGen).Map(func(args interface{}) string {
		av := args.([]interface{})
		return filepath.ToSlash(filepath.Join(av[0].(string), av[1].(string)))
	})
}

func genPathComponent(containsDot bool) gopter.Gen {
	elementGen := gen.OneGenOf(
		gen.AlphaNumChar(),
		gen.RuneRange(0x20, 0x40),
	).SuchThat(func(arg interface{}) bool {
		ch := arg.(rune)
		if ch == '\\' || ch == '/' || ch == ':' {
			return false
		}
		if !containsDot && ch == '.' {
			return false
		}
		return true
	})
	runeToString := func(r []rune) string {
		return string(r)
	}
	return gen.SliceOf(elementGen).Map(runeToString).SuchThat(func(arg string) bool { return 0 < len(arg) })
}

func genVolume() gopter.Gen {
	genDrive := gen.AlphaUpperChar().Map(func(arg interface{}) string {
		ch := arg.(rune)
		return fmt.Sprintf("%c:", ch)
	}).WithLabel("drive")
	genShare := gopter.CombineGens(gen.Identifier(), gen.Identifier()).Map(func(args interface{}) string {
		a := args.([]interface{})
		a0 := a[0].(string)
		a1 := a[1].(string)
		return fmt.Sprintf(`\\%s\%s`, a0, a1)
	}).WithLabel("share")
	return gen.OneGenOf(genDrive, genShare)
}

// To replace `.txt` to `.bin`
func ExampleReplaceExtension() {
	const f = "foo.bar.txt"
	fmt.Println(ReplaceExtension(f, ".bin"))
	// Output: foo.bar.bin
}

func ExampleBasename() {
	const f = "foo/bar/baz.qux"
	fmt.Println(Basename(f))
	fmt.Println(Basename(f, "qux"))
	// Output:
	// baz.qux
	// baz
}
