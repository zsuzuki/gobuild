package main

import (
	"testing"

	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/yaml.v2"
)

func TestUnmarshalStringList(t *testing.T) {
	srcYAML := `
type: PS4
target: foo
list:
- list item
debug:
- debug item
release:
- release item
develop:
- develop item
develop-release:
- develop-release item
product:
- product item
`
	Convey(`Test UnmarshalStringList`, t, func() {
		var slist StringList
		err := yaml.Unmarshal([]byte(srcYAML), &slist)
		So(err, ShouldBeNil)
		Convey(`Test slots...`, func() {
			Convey(`Test fixed slots.`, func() {
				So(slist.Type, ShouldEqual, "PS4")
				So(slist.Target, ShouldEqual, "foo")
			})
			for _, k := range KnownBuildTypes {
				Convey(fmt.Sprintf(`Test "%s" slot.`, k.String()), func() {
					Convey(`Passed as KnownBuildType`, func() {
						l := slist.Items(k)
						So(*l, ShouldNotBeEmpty)
						So((*l)[0], ShouldEqual, fmt.Sprintf("%s item", k.String()))
					})
					Convey(`Passed as string`, func() {
						key := k.String()
						l := slist.Items(key)
						So(*l, ShouldNotBeEmpty)
						So((*l)[0], ShouldEqual, fmt.Sprintf("%s item", key))
					})
				})
			}
			Convey(`Unexisted key`, func() {
				l := slist.Items("foo")
				So(l, ShouldBeNil)
			})
		})
	})
}
