package main

import (
	"testing"

	"github.com/leanovate/gopter/arbitrary"
	"github.com/leanovate/gopter/convey"
	"github.com/leanovate/gopter/gen"
	"gopkg.in/yaml.v2"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPlatformID_Simple(t *testing.T) {
	Convey(`GIVEN: An empty PlatformIDSet`, t, func() {
		var tmp PlatformIDSet
		Convey(`WHEN: Add some values`, func() {
			tmp.Add("abc")
			tmp.Add("def")
			Convey(`AND WHEN: Marshal it`, func() {
				b, err := yaml.Marshal(&tmp)
				t.Logf("b = %s", string(b))
				Convey(`THEN: Should success`, func() {
					So(err, ShouldBeNil)
					Convey(`AND WHEN: Unmarshal it`, func() {
						var vv PlatformIDSet
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
		var result PlatformIDSet
		for _, v := range arg.([]string) {
			result.Add(PlatformID(v))
		}
		return &result
	}))
	condition := func(platforms *PlatformIDSet) bool {
		b, err := yaml.Marshal(platforms)
		if err != nil {
			t.Logf("%v", err)
			return false
		}
		t.Logf(string(b))
		var vv PlatformIDSet
		err = yaml.Unmarshal(b, &vv)
		if err != nil {
			t.Logf("%v", err)
			return false
		}
		return platforms.Equals(vv)
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
			var v PlatformIDSet
			err := yaml.Unmarshal(([]byte)(src), &v)
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
			var v PlatformIDSet
			err := yaml.Unmarshal(([]byte)(src), &v)
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
