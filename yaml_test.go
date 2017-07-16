package main

import (
	"fmt"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/leanovate/gopter/arbitrary"
	"github.com/leanovate/gopter/convey"
	"github.com/leanovate/gopter/gen"

	. "github.com/smartystreets/goconvey/convey"
)

const (
	platformMac     PlatformId = "Mac"
	platformWindows PlatformId = "WIN"
	platformLinux   PlatformId = "LINUX"
)

func TestUnmarshalStringList(t *testing.T) {
	srcYAML := `# YAML Source
type: [WIN, LINUX, Mac]
target: foo
list:
- list item
- dummy
debug:
- debug item
- dummy
release:
- release item
- dummy
develop:
- develop item
- dummy
develop-release:
- develop-release item
- dummy
product:
- product item
- dummy
`
	Convey(`GIVEN: A YAML source`, t, func() {
		Convey("WHEN: Unmarshal", func() {
			var slist StringList
			err := yaml.Unmarshal([]byte(srcYAML), &slist)
			Convey("THEN: Should success", func() {
				So(err, ShouldBeNil)
				Convey("AND THEN: .Type should contain \"Mac\"", func() {
					So(slist.Types(), ShouldContain, platformMac)
				})
				Convey("AND THEN: .Target should be \"foo\"", func() {
					So(slist.Target, ShouldEqual, "foo")
				})
				Convey(`AND WHEN: Call Items ("foo")`, func() {
					l := slist.Items("foo")
					Convey("THEN: Should return `nil`", func() {
						So(l, ShouldBeNil)
					})
				})
				Convey("AND WHEN: Call `Item` with `KnownBuildType` key", func() {
					for _, k := range KnownBuildTypes {
						Convey(fmt.Sprintf(`AND WHEN: Call Item ("%s")`, k.String()), func() {
							l := slist.Items(k)
							Convey("AND THEN: Should not return nil", func() {
								So(l, ShouldNotBeNil)
								Convey(fmt.Sprintf("AND THEN: Should return [\"%s item\", \"dummy\"]", k.String()), func() {
									So(*l, ShouldResemble, []string{fmt.Sprintf("%s item", k.String()), "dummy"})
								})
							})
						})
					}
				})
				Convey("AND WHEN: Call `Item` with `string` key", func() {
					for _, k := range KnownBuildTypes {
						Convey(fmt.Sprintf(`AND WHEN: Call Item ("%s")`, k.String()), func() {
							key := k.String()
							l := slist.Items(key)
							Convey("AND THEN: Should not return nil", func() {
								So(l, ShouldNotBeNil)
								Convey(fmt.Sprintf("AND THEN: Should return [\"%s item\", \"dummy\"]", key), func() {
									So(*l, ShouldResemble, []string{fmt.Sprintf("%s item", key), "dummy"})
								})
							})
						})
					}
				})
			})
		})
	})
}

func TestStringList_UnmarshalYAML_Type(t *testing.T) {
	Convey(`Test unmarshaler type slot`, t, func() {
		Convey(`GIVEN: YAML with no types`, func() {
			srcYAML := `
target: foo
list:
- item1
- dummy`
			Convey(`WHEN: Unmarshal`, func() {
				var slist StringList
				err := yaml.Unmarshal([]byte(srcYAML), &slist)
				Convey(`THEN: Should success`, func() {
					So(err, ShouldBeNil)
					Convey(`AND THEN: Type should be []`, func() {
						So(slist.Types(), ShouldBeEmpty)
					})
					Convey(`AND THEN: list should be ["item1", "dummy"]`, func() {
						So(*slist.Items("list"), ShouldResemble, []string{"item1", "dummy"})
					})
				})
			})
		})
		Convey(`GIVEN: YAML with single type`, func() {
			srcYAML := `
type: Mac
target: foo
list:
- item1
- dummy`
			Convey(`WHEN: Unmarshal`, func() {
				var slist StringList
				err := yaml.Unmarshal([]byte(srcYAML), &slist)
				Convey(`THEN: Should success`, func() {
					So(err, ShouldBeNil)
					Convey(`AND THEN: Type should be ["Mac"]`, func() {
						So(slist.Types(), ShouldResemble, []PlatformId{platformMac})
					})
					Convey(`AND THEN: list should be ["item1", "dummy"]`, func() {
						So(*slist.Items("list"), ShouldResemble, []string{"item1", "dummy"})
					})
				})
			})
		})
		Convey(`GIVEN: YAML with multiple types`, func() {
			srcYAML := `
type: [Mac, WIN, LINUX]
target: foo
list:
- item1-2
- dummy-2`
			Convey(`WHEN: Unmarshal`, func() {
				var slist StringList
				err := yaml.Unmarshal([]byte(srcYAML), &slist)
				Convey(`THEN: Should success`, func() {
					So(err, ShouldBeNil)
					Convey(`AND THEN: Type should be ["Mac", "WIN", "LINUX"]`, func() {
						So(slist.Types(), ShouldResemble, []PlatformId{platformMac, platformWindows, platformLinux})
					})
					Convey(`AND THEN: list should be ["item1-2", "dummy-2"]`, func() {
						So(*slist.Items("list"), ShouldResemble, []string{"item1-2", "dummy-2"})
					})
				})
			})
		})
	})
}

func TestVariable(t *testing.T) {
	arbitraries := arbitrary.DefaultArbitraries()
	arbitraries.RegisterGen(gen.Identifier().Map(func(arg interface{}) PlatformId {
		v := arg.(string)
		return PlatformId(v)
	}))
	condition := func(name string, value string, platform PlatformId, target string, build string) bool {
		v := Variable{
			Name:     name,
			Value:    value,
			Platform: platform,
			Target:   target,
			Build:    build,
		}
		b, err := yaml.Marshal(&v)
		if err != nil {
			t.Logf("%v", err)
			return false
		}
		var vv Variable
		err = yaml.Unmarshal(b, &vv)
		if err != nil {
			t.Logf("%v", err)
			return false
		}
		//t.Logf("Platform: %s", vv.Platform.String())
		return v == vv
	}
	Convey(`Marshal then Unmarshal should return to original`, t, func() {
		So(condition, convey.ShouldSucceedForAll, arbitraries)
	})
}

//func TestIntParse(t *testing.T) {
//  properties := gopter.NewProperties(nil)
//  arbitraries := arbitrary.DefaultArbitraries()
//
//  properties.Property("printed integers can be parsed", arbitraries.ForAll(
//		func(a int64) bool {
//			str := fmt.Sprintf("%d", a)
//			parsed, err := strconv.ParseInt(str, 10, 64)
//			return err == nil && parsed == a
//		}))
//
//  properties.TestingRun(t)
//}
