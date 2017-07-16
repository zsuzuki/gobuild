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
	platformMac     PlatformID = "Mac"
	platformWindows PlatformID = "WIN"
	platformLinux   PlatformID = "LINUX"
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
						So(slist.Types(), ShouldResemble, []PlatformID{platformMac})
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
						So(slist.Types(), ShouldResemble, []PlatformID{platformMac, platformWindows, platformLinux})
					})
					Convey(`AND THEN: list should be ["item1-2", "dummy-2"]`, func() {
						So(*slist.Items("list"), ShouldResemble, []string{"item1-2", "dummy-2"})
					})
				})
			})
		})
	})
}

func TestPlatformID_Simple(t *testing.T) {
	Convey(`GIVEN: An empty PlatformIDSet`, t, func() {
		tmp := NewPlatformIDSet()
		Convey(`WHEN: Add some values`, func() {
			tmp.Add("abc")
			tmp.Add("def")
			Convey(`AND WHEN: Marshal it`, func() {
				b, err := yaml.Marshal(tmp)
				t.Logf("b = %s", string(b))
				Convey(`THEN: Should success`, func() {
					So(err, ShouldBeNil)
					Convey(`AND WHEN: Unmarshal it`, func() {
						vv := NewPlatformIDSet()
						err = yaml.Unmarshal(b, &vv)
						Convey(`THEN: Should success`, func() {
							So(err, ShouldBeNil)
							Convey(`AND THEN: Contains added values`, func() {
								So(vv.Contains("abc"), ShouldBeTrue)
								So(vv.Contains("def"), ShouldBeTrue)
							})
						})
					})
				})
			})
		})
	})
}

func TestPlatformIDSet_MarshalThenUnmarshal(t *testing.T) {
	arbitraries := arbitrary.DefaultArbitraries()
	arbitraries.RegisterGen(gen.Identifier().Map(func(arg interface{}) PlatformID {
		v := arg.(string)
		return PlatformID(v)
	}))
	arbitraries.RegisterGen(gen.SliceOf(gen.Identifier()).Map(func(arg interface{}) *PlatformIDSet {
		result := NewPlatformIDSet()
		for _, v := range arg.([]string) {
			result.Add(PlatformID(v))
		}
		return result
	}))
	condition := func(platforms *PlatformIDSet) bool {
		b, err := yaml.Marshal(platforms)
		if err != nil {
			t.Logf("%v", err)
			return false
		}
		t.Logf(string(b))
		vv := new(PlatformIDSet)
		err = yaml.Unmarshal(b, vv)
		if err != nil {
			t.Logf("%v", err)
			return false
		}
		return platforms.Equals(*vv)
	}
	Convey(`Perform Marshal then Unmarshal return to original`, t, func() {
		So(condition, convey.ShouldSucceedForAll, arbitraries)
	})
}

func TestPlatformIDSet_UnmarshalYAML(t *testing.T) {
	Convey(`GIVEN: YAML Source`, t, func() {
		src := `
LINUX
`
		Convey(`WHEN: Unmarshal it`, func() {
			v := NewPlatformIDSet()
			err := yaml.Unmarshal(([]byte)(src), v)
			Convey(`THEN: Should success`, func() {
				So(err, ShouldBeNil)
				Convey(`AND WHEN: Should contain LINUX`, func() {
					So(v.Contains("LINUX"), ShouldBeTrue)
				})
			})
		})
	})
	Convey(`GIVEN: YAML Source`, t, func() {
		src := `
[LINUX, Windows, BeOS]
`
		Convey(`WHEN: Unmarshal it`, func() {
			v := NewPlatformIDSet()
			err := yaml.Unmarshal(([]byte)(src), v)
			Convey(`THEN: Should success`, func() {
				So(err, ShouldBeNil)
				Convey(`AND WHEN: Should contain LINUX, Windows and BeOS`, func() {
					So(v.Contains("LINUX"), ShouldBeTrue)
					So(v.Contains("Windows"), ShouldBeTrue)
					So(v.Contains("BeOS"), ShouldBeTrue)
				})
			})
		})
	})
}

func TestVariable(t *testing.T) {
	arbitraries := arbitrary.DefaultArbitraries()
	arbitraries.RegisterGen(gen.Identifier().Map(func(arg interface{}) PlatformID {
		v := arg.(string)
		return PlatformID(v)
	}))
	arbitraries.RegisterGen(gen.SliceOf(gen.Identifier()).Map(func(arg interface{}) map[PlatformID]bool {
		result := make(map[PlatformID]bool)
		for _, v := range arg.([]string) {
			result[PlatformID(v)] = true
		}
		return result
	}))
	condition := func(name string, value string, platforms map[PlatformID]bool, target string, build string) bool {
		v := Variable{
			Name:      name,
			Value:     value,
			platforms: platforms,
			Target:    target,
			Build:     build,
		}
		b, err := yaml.Marshal(&v)
		if err != nil {
			t.Logf("%v", err)
			return false
		}
		// t.Logf("%s\n", string(b))
		var vv Variable
		err = yaml.Unmarshal(b, &vv)
		if err != nil {
			t.Logf("%v", err)
			return false
		}
		//t.Logf("Platform: %v", vv.platforms)
		return v.Equals(&vv)
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
