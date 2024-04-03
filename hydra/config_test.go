package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_ValueSwitchNil(t *testing.T) {

	Convey("When a ValueSwitch is passed a nil interface{}, it returns -1", t, func() {
		var v interface{}

		So(ValueSwitch(v), ShouldEqual, -1)

	})
}

func Test_ValueSwitchString(t *testing.T) {

	Convey("When a ValueSwitch is passed a string, it returns -1", t, func() {
		var v interface{} = "Hello World"

		So(ValueSwitch(v), ShouldEqual, -1)

	})
}

func Test_ValueSwitchInt(t *testing.T) {

	Convey("When a ValueSwitch is passed an int, it returns the int (values -100 to 100 tested)", t, func() {
		var v interface{}

		for i := -100; i <= 100; i++ {
			v = i
			So(ValueSwitch(v), ShouldEqual, i)
		}

	})
}

func Test_ValueBomb(t *testing.T) {

	Convey("When a ValueBomb is passed a negative and positive ints, it returns properly (values -100 to 100 tested)", t, func() {
		var v interface{}

		for i := -100; i <= 100; i++ {
			v = i

			vu, vok := ValueBomb(v, true)
			if i < 0 {
				So(vok, ShouldBeFalse)
			} else {
				// positive
				So(vok, ShouldBeTrue)
				So(vu, ShouldEqual, i)
			}
		}

	})
}
