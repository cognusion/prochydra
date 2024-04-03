package head

import (
	"github.com/fortytw2/leaktest"
	. "github.com/smartystreets/goconvey/convey"

	"bytes"
	"log"
	"testing"
)

func Test_BufioHuge(t *testing.T) {
	defer leaktest.Check(t)()

	M1 := make([]byte, 100000000)
	for i := range M1 {
		M1[i] = byte(26)
	}
	buf := bytes.NewBuffer(M1)
	var bufw Sbuffer

	Convey("A bufio.Scanner is wrapped around our massive buffer", t, func() {
		w := log.New(&bufw, "", 0)
		errorChan := make(chan error, 1)

		ReadLogger(buf, w, errorChan)

		var e error
		select {
		case e = <-errorChan:

		default:
		}

		So(e, ShouldBeNil)
	})
}
