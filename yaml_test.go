package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnmarshalStringList(t *testing.T) {
	srcYAML := `# YAML Source
type: [WIN, LINUX, PS4]
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
				Convey("AND THEN: .Type should contain \"PS4\"", func() {
					So(slist.Types(), ShouldContain, "PS4")
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
		Convey(`GIVEN: YAML with single type`, func() {
			srcYAML := `
type: PS4
target: foo
list:
- item1
- dummy`
			Convey(`WHEN: Unmarshal`, func() {
				var slist StringList
				err := yaml.Unmarshal([]byte(srcYAML), &slist)
				Convey(`THEN: Should success`, func() {
					So(err, ShouldBeNil)
					Convey(`AND THEN: Type should be ["PS4"]`, func() {
						So(slist.Types(), ShouldResemble, []string{"PS4"})
					})
					Convey(`AND THEN: list should be ["item1", "dummy"]`, func() {
						So(*slist.Items("list"), ShouldResemble, []string{"item1", "dummy"})
					})
				})
			})
		})
		Convey(`GIVEN: YAML with multiple types`, func() {
			srcYAML := `
type: [PS4, WIN, LINUX]
target: foo
list:
- item1-2
- dummy-2`
			Convey(`WHEN: Unmarshal`, func() {
				var slist StringList
				err := yaml.Unmarshal([]byte(srcYAML), &slist)
				Convey(`THEN: Should success`, func() {
					So(err, ShouldBeNil)
					Convey(`AND THEN: Type should be ["PS4", "WIN", "LINUX"]`, func() {
						So(slist.Types(), ShouldResemble, []string{"PS4", "WIN", "LINUX"})
					})
					Convey(`AND THEN: list should be ["item1-2", "dummy-2"]`, func() {
						So(*slist.Items("list"), ShouldResemble, []string{"item1-2", "dummy-2"})
					})
				})
			})
		})
	})
}
