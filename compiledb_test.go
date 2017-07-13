package main

import (
	"bytes"
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestWriteCompileDbWithEmptyDefs(t *testing.T) {
	Convey("GIVEN: Empty definiton", t, func() {
		src := make([]CompileDbItem, 0)
		Convey("WHEN: Call `WriteCompileDb`", func() {
			var buf bytes.Buffer
			err := WriteCompileDb(&buf, src)
			Convey("THEN: Should success", func() {
				So(err, ShouldBeNil)
				Convey("AND THEN: Written empty array", func() {
					var actual []CompileDbItem
					err := json.Unmarshal(buf.Bytes(), &actual)
					So(err, ShouldBeNil)
					So(actual, ShouldBeEmpty)
				})
			})
		})
	})
}

func TestWriteCompileDb(t *testing.T) {
	Convey("GIVEN: Definitions", t, func() {
		src := make([]CompileDbItem, 0)
		item := CompileDbItem{
			File:      "foo",
			Directory: "foo.dir",
			Output:    "foo.output",
			Arguments: []string{"a0", "a1", "a2"},
		}
		src = append(src, item)
		Convey("WHEN: Call `WriteCompileDb` with 1 item", func() {
			var buf bytes.Buffer
			err := WriteCompileDb(&buf, src)
			Convey("THEN: Should success", func() {
				So(err, ShouldBeNil)
				Convey("AND THEN: Written empty array", func() {
					var actual []CompileDbItem
					err := json.Unmarshal(buf.Bytes(), &actual)
					So(err, ShouldBeNil)
					So(actual, ShouldHaveLength, 1)
					So(actual, ShouldResemble, src)
				})
			})
		})
		item2 := CompileDbItem{
			File:      "bar",
			Directory: "bar.dir",
			Output:    "bar.output",
			Arguments: []string{"a0", "a1", "a2"},
		}
		src = append(src, item2)
		Convey("WHEN: Call `WriteCompileDb` with 2 items", func() {
			var buf bytes.Buffer
			err := WriteCompileDb(&buf, src)
			Convey("THEN: Should success", func() {
				So(err, ShouldBeNil)
				Convey("AND THEN: Written empty array", func() {
					var actual []CompileDbItem
					err := json.Unmarshal(buf.Bytes(), &actual)
					So(err, ShouldBeNil)
					So(actual, ShouldHaveLength, 2)
					So(actual, ShouldResemble, src)
				})
			})
		})
	})
}
