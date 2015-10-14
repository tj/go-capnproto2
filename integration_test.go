package capnp_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	cv "github.com/smartystreets/goconvey/convey"
	capnp "zombiezen.com/go/capnproto2"
	air "zombiezen.com/go/capnproto2/internal/aircraftlib"
)

type bitListTest struct {
	list []bool
	text string
}

var bitListTests = []bitListTest{
	{
		[]bool{true, false, true},
		"(boolvec = [true, false, true])\n",
	},
	{
		[]bool{false},
		"(boolvec = [false])\n",
	},
	{
		[]bool{true},
		"(boolvec = [true])\n",
	},
	{
		[]bool{false, true},
		"(boolvec = [false, true])\n",
	},
	{
		[]bool{true, true},
		"(boolvec = [true, true])\n",
	},
	{
		[]bool{false, false, true},
		"(boolvec = [false, false, true])\n",
	},
	{
		[]bool{true, false, true, false, true},
		"(boolvec = [true, false, true, false, true])\n",
	},
	{
		[]bool{
			false, false, false, false, false, false, false, false,
			false, false, false, false, false, false, false, false,
			false, false, false, false, false, false, false, false,
			false, false, false, false, false, false, false, false,
			false, false, false, false, false, false, false, false,
			false, false, false, false, false, false, false, false,
			false, false, false, false, false, false, false, false,
			false, false, false, false, false, false, false, false,
			true, true,
		},
		"(boolvec = [false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true])\n",
	},
}

func (blt bitListTest) makeMessage() (*capnp.Message, error) {
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}
	z, err := air.NewRootZ(seg)
	if err != nil {
		return nil, err
	}
	list, err := capnp.NewBitList(seg, int32(len(blt.list)))
	if err != nil {
		return nil, err
	}
	for i := range blt.list {
		list.Set(i, blt.list[i])
	}
	if err := z.SetBoolvec(list); err != nil {
		return nil, err
	}
	return msg, nil
}

func TestBitList(t *testing.T) {
	for _, test := range bitListTests {
		msg, err := test.makeMessage()
		if err != nil {
			t.Errorf("%v: make message: %v", test.list, err)
			continue
		}

		z, err := air.ReadRootZ(msg)
		if err != nil {
			t.Errorf("%v: read root Z: %v", test.list, err)
			continue
		}
		if w := z.Which(); w != air.Z_Which_boolvec {
			t.Errorf("%v: root.Which() = %v; want boolvec", test.list, w)
			continue
		}
		list, err := z.Boolvec()
		if err != nil {
			t.Errorf("%v: read Z.boolvec: %v", test.list, err)
			continue
		}
		if n := list.Len(); n != len(test.list) {
			t.Errorf("%v: len(Z.boolvec) = %d; want %d", test.list, n, len(test.list))
			continue
		}
		for i := range test.list {
			if li := list.At(i); li != test.list[i] {
				t.Errorf("%v: Z.boolvec[%d] = %t; want %t", test.list, i, li, test.list[i])
			}
		}
	}
}

func TestBitList_Decode(t *testing.T) {
	// TODO(light): skip if tool not present
	for _, test := range bitListTests {
		msg, err := test.makeMessage()
		if err != nil {
			t.Errorf("%v: make message: %v", test.list, err)
			continue
		}
		seg, _ := msg.Segment(0)
		text := CapnpDecodeSegment(seg, "", schemaPath, "Z")
		// TODO(light): don't trim
		if want := strings.TrimSpace(test.text); text != want {
			t.Errorf("%v: capnp decode = %q; want %q", test.list, text, want)
		}
	}
}

// A marshalTest tests whether a message can be encoded then read by the
// reference capnp implementation.
type marshalTest struct {
	name string

	msg *capnp.Message
	typ string

	text string
	data []byte
}

func makeMarshalTests(t *testing.T) []marshalTest {
	tests := []marshalTest{
		{
			name: "zdateFilledMessage(1)",
			msg:  zdateFilledMessage(t, 1),
			typ:  "Z",
			text: "(zdatevec = [(year = 2004, month = 12, day = 7)])\n",
		},
		{
			name: "zdateFilledMessage(10)",
			msg:  zdateFilledMessage(t, 10),
			typ:  "Z",
			text: "(zdatevec = [(year = 2004, month = 12, day = 7), (year = 2005, month = 12, day = 7), (year = 2006, month = 12, day = 7), (year = 2007, month = 12, day = 7), (year = 2008, month = 12, day = 7), (year = 2009, month = 12, day = 7), (year = 2010, month = 12, day = 7), (year = 2011, month = 12, day = 7), (year = 2012, month = 12, day = 7), (year = 2013, month = 12, day = 7)])\n",
		},
		{
			name: "zdataFilledMessage(20)",
			msg:  zdataFilledMessage(t, 20),
			typ:  "Z",
			text: `(zdata = (data = "\x00\x01\x02\x03\x04\x05\x06\a\b\t\n\v\f\r\x0e\x0f\x10\x11\x12\x13"))` + "\n",
			data: []byte{
				0, 0, 0, 0, 8, 0, 0, 0,
				0, 0, 0, 0, 2, 0, 1, 0,
				28, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 1, 0,
				1, 0, 0, 0, 162, 0, 0, 0,
				0, 1, 2, 3, 4, 5, 6, 7,
				8, 9, 10, 11, 12, 13, 14, 15,
				16, 17, 18, 19, 0, 0, 0, 0,
			},
		},
	}

	{
		msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := air.NewRootZjob(seg); err != nil {
			t.Fatal(err)
		}
		tests = append(tests, marshalTest{
			name: "empty Zjob",
			msg:  msg,
			typ:  "Zjob",
			text: "()\n",
			data: []byte{
				0, 0, 0, 0, 3, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 2, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
			},
		})
	}

	{
		msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			t.Fatal(err)
		}
		zjob, err := air.NewRootZjob(seg)
		if err != nil {
			t.Fatal(err)
		}
		if err := zjob.SetCmd("abc"); err != nil {
			t.Fatal(err)
		}
		tests = append(tests, marshalTest{
			name: "Zjob with text",
			msg:  msg,
			typ:  "Zjob",
			text: "(cmd = \"abc\")\n",
			data: []byte{
				0, 0, 0, 0, 4, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 2, 0,
				5, 0, 0, 0, 34, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				97, 98, 99, 0, 0, 0, 0, 0,
			},
		})
	}

	{
		msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			t.Fatal(err)
		}
		zjob, err := air.NewRootZjob(seg)
		if err != nil {
			t.Fatal(err)
		}
		tl, err := capnp.NewTextList(seg, 1)
		if err != nil {
			t.Fatal(err)
		}
		if err := tl.Set(0, "xyz"); err != nil {
			t.Fatal(err)
		}
		if err := zjob.SetArgs(tl); err != nil {
			t.Fatal(err)
		}
		tests = append(tests, marshalTest{
			name: "Zjob with text list",
			msg:  msg,
			typ:  "Zjob",
			text: "(args = [\"xyz\"])\n",
			data: []byte{
				0, 0, 0, 0, 5, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 2, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				1, 0, 0, 0, 14, 0, 0, 0,
				1, 0, 0, 0, 34, 0, 0, 0,
				120, 121, 122, 0, 0, 0, 0, 0,
			},
		})
	}

	{
		msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			t.Fatal(err)
		}
		zjob, err := air.NewRootZjob(seg)
		if err != nil {
			t.Fatal(err)
		}
		if err := zjob.SetCmd("abc"); err != nil {
			t.Fatal(err)
		}
		tl, err := capnp.NewTextList(seg, 1)
		if err != nil {
			t.Fatal(err)
		}
		if err := tl.Set(0, "xyz"); err != nil {
			t.Fatal(err)
		}
		if err := zjob.SetArgs(tl); err != nil {
			t.Fatal(err)
		}
		tests = append(tests, marshalTest{
			name: "Zjob with text and text list",
			msg:  msg,
			typ:  "Zjob",
			text: "(cmd = \"abc\", args = [\"xyz\"])\n",
			data: []byte{
				0, 0, 0, 0, 6, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 2, 0,
				5, 0, 0, 0, 34, 0, 0, 0,
				5, 0, 0, 0, 14, 0, 0, 0,
				97, 98, 99, 0, 0, 0, 0, 0,
				1, 0, 0, 0, 34, 0, 0, 0,
				120, 121, 122, 0, 0, 0, 0, 0,
			},
		})
	}

	{
		msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			t.Fatal(err)
		}
		server, err := air.NewRootZserver(seg)
		if err != nil {
			t.Fatal(err)
		}
		joblist, err := air.NewZjob_List(seg, 1)
		if err != nil {
			t.Fatal(err)
		}
		if err := server.SetWaitingjobs(joblist); err != nil {
			t.Fatal(err)
		}

		tests = append(tests, marshalTest{
			name: "Zserver with one empty job",
			msg:  msg,
			typ:  "Zserver",
			text: "(waitingjobs = [()])\n",
			data: []byte{
				0, 0, 0, 0, 5, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 1, 0,
				1, 0, 0, 0, 23, 0, 0, 0,
				4, 0, 0, 0, 0, 0, 2, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
			},
		})
	}

	{
		msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			t.Fatal(err)
		}
		server, err := air.NewRootZserver(seg)
		if err != nil {
			t.Fatal(err)
		}
		joblist, err := air.NewZjob_List(seg, 1)
		if err != nil {
			t.Fatal(err)
		}
		server.SetWaitingjobs(joblist)
		if err := joblist.At(0).SetCmd("abc"); err != nil {
			t.Fatal(err)
		}
		tl, err := capnp.NewTextList(seg, 1)
		if err != nil {
			t.Fatal(err)
		}
		if err := tl.Set(0, "xyz"); err != nil {
			t.Fatal(err)
		}
		if err := joblist.At(0).SetArgs(tl); err != nil {
			t.Fatal(err)
		}

		tests = append(tests, marshalTest{
			name: "Zserver with one full job",
			msg:  msg,
			typ:  "Zserver",
			text: "(waitingjobs = [(cmd = \"abc\", args = [\"xyz\"])])\n",
			data: []byte{
				0, 0, 0, 0, 8, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 1, 0,
				1, 0, 0, 0, 23, 0, 0, 0,
				4, 0, 0, 0, 0, 0, 2, 0,
				5, 0, 0, 0, 34, 0, 0, 0,
				5, 0, 0, 0, 14, 0, 0, 0,
				97, 98, 99, 0, 0, 0, 0, 0,
				1, 0, 0, 0, 34, 0, 0, 0,
				120, 121, 122, 0, 0, 0, 0, 0,
			},
		})
	}

	{
		msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			t.Fatal(err)
		}
		server, err := air.NewRootZserver(seg)
		if err != nil {
			t.Fatal(err)
		}
		joblist, err := air.NewZjob_List(seg, 2)
		if err != nil {
			t.Fatal(err)
		}
		server.SetWaitingjobs(joblist)
		if err := joblist.At(0).SetCmd("abc"); err != nil {
			t.Fatal(err)
		}
		if err := joblist.At(1).SetCmd("xyz"); err != nil {
			t.Fatal(err)
		}

		tests = append(tests, marshalTest{
			name: "Zserver with two jobs",
			msg:  msg,
			typ:  "Zserver",
			text: "(waitingjobs = [(cmd = \"abc\"), (cmd = \"xyz\")])\n",
			data: []byte{
				0, 0, 0, 0, 9, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 1, 0,
				1, 0, 0, 0, 39, 0, 0, 0,
				8, 0, 0, 0, 0, 0, 2, 0,
				13, 0, 0, 0, 34, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				9, 0, 0, 0, 34, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				97, 98, 99, 0, 0, 0, 0, 0,
				120, 121, 122, 0, 0, 0, 0, 0,
			},
		})
	}

	{
		msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			t.Fatal(err)
		}
		_, scratch, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			t.Fatal(err)
		}

		// in seg
		segbag, err := air.NewRootBag(seg)
		if err != nil {
			t.Fatal(err)
		}

		// in scratch
		xc, err := air.NewRootCounter(scratch)
		if err != nil {
			t.Fatal(err)
		}
		xc.SetSize(9)

		// copy from scratch to seg
		if err = segbag.SetCounter(xc); err != nil {
			t.Fatal(err)
		}

		tests = append(tests, marshalTest{
			name: "copy struct between messages",
			msg:  msg,
			typ:  "Bag",
			text: "(counter = (size = 9))\n",
			data: []byte{
				0, 0, 0, 0, 5, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 1, 0,
				0, 0, 0, 0, 1, 0, 2, 0,
				9, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
			},
		})
	}

	{
		msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			t.Fatal(err)
		}
		_, scratch, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			t.Fatal(err)
		}

		// in seg
		segbag, err := air.NewRootBag(seg)
		if err != nil {
			t.Fatal(err)
		}

		// in scratch
		xc, err := air.NewRootCounter(scratch)
		if err != nil {
			t.Fatal(err)
		}
		xc.SetSize(9)
		if err := xc.SetWords("hello"); err != nil {
			t.Fatal(err)
		}

		// copy from scratch to seg
		if err = segbag.SetCounter(xc); err != nil {
			t.Fatal(err)
		}

		tests = append(tests, marshalTest{
			name: "copy struct with text between messages",
			msg:  msg,
			typ:  "Bag",
			text: "(counter = (size = 9, words = \"hello\"))\n",
			data: []byte{
				0, 0, 0, 0, 6, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 1, 0,
				0, 0, 0, 0, 1, 0, 2, 0,
				9, 0, 0, 0, 0, 0, 0, 0,
				5, 0, 0, 0, 50, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				104, 101, 108, 108, 111, 0, 0, 0,
			},
		})
	}

	{
		msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			t.Fatal(err)
		}
		_, scratch, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			t.Fatal(err)
		}

		// in seg
		segbag, err := air.NewRootBag(seg)
		if err != nil {
			t.Fatal(err)
		}

		// in scratch
		xc, err := air.NewRootCounter(scratch)
		if err != nil {
			t.Fatal(err)
		}
		xc.SetSize(9)
		tl, err := capnp.NewTextList(scratch, 2)
		if err != nil {
			t.Fatal(err)
		}
		if err := tl.Set(0, "hello"); err != nil {
			t.Fatal(err)
		}
		if err := tl.Set(1, "bye"); err != nil {
			t.Fatal(err)
		}
		if err := xc.SetWordlist(tl); err != nil {
			t.Fatal(err)
		}

		// copy from scratch to seg
		if err = segbag.SetCounter(xc); err != nil {
			t.Fatal(err)
		}

		tests = append(tests, marshalTest{
			name: "copy struct with list of text between messages",
			msg:  msg,
			typ:  "Bag",
			text: "(counter = (size = 9, wordlist = [\"hello\", \"bye\"]))\n",
			data: []byte{
				0, 0, 0, 0, 9, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 1, 0,
				0, 0, 0, 0, 1, 0, 2, 0,
				9, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				1, 0, 0, 0, 22, 0, 0, 0,
				5, 0, 0, 0, 50, 0, 0, 0,
				5, 0, 0, 0, 34, 0, 0, 0,
				104, 101, 108, 108, 111, 0, 0, 0,
				98, 121, 101, 0, 0, 0, 0, 0,
			},
		})
	}

	{
		msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			t.Fatal(err)
		}
		_, scratch, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			t.Fatal(err)
		}

		// in seg
		segbag, err := air.NewRootBag(seg)
		if err != nil {
			t.Fatal(err)
		}

		// in scratch
		xc, err := air.NewRootCounter(scratch)
		if err != nil {
			t.Fatal(err)
		}
		xc.SetSize(9)
		if err := xc.SetWords("abc"); err != nil {
			t.Fatal(err)
		}
		tl, err := capnp.NewTextList(scratch, 2)
		if err != nil {
			t.Fatal(err)
		}
		if err := tl.Set(0, "hello"); err != nil {
			t.Fatal(err)
		}
		if err := tl.Set(1, "byenow"); err != nil {
			t.Fatal(err)
		}
		if err := xc.SetWordlist(tl); err != nil {
			t.Fatal(err)
		}

		// copy from scratch to seg
		if err = segbag.SetCounter(xc); err != nil {
			t.Fatal(err)
		}

		tests = append(tests, marshalTest{
			name: "copy struct with data, text, and list of text between messages",
			msg:  msg,
			typ:  "Bag",
			text: "(counter = (size = 9, words = \"abc\", wordlist = [\"hello\", \"byenow\"]))\n",
			data: []byte{
				0, 0, 0, 0, 10, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 1, 0,
				0, 0, 0, 0, 1, 0, 2, 0,
				9, 0, 0, 0, 0, 0, 0, 0,
				5, 0, 0, 0, 34, 0, 0, 0,
				5, 0, 0, 0, 22, 0, 0, 0,
				97, 98, 99, 0, 0, 0, 0, 0,
				5, 0, 0, 0, 50, 0, 0, 0,
				5, 0, 0, 0, 58, 0, 0, 0,
				104, 101, 108, 108, 111, 0, 0, 0,
				98, 121, 101, 110, 111, 119, 0, 0,
			},
		})
	}

	{
		msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			t.Fatal(err)
		}
		holder, err := air.NewRootHoldsVerEmptyList(seg)
		if err != nil {
			t.Fatal(err)
		}
		elist, err := air.NewVerEmpty_List(seg, 2)
		if err != nil {
			t.Fatal(err)
		}
		if err := holder.SetMylist(elist); err != nil {
			t.Fatal(err)
		}

		tests = append(tests, marshalTest{
			name: "V0 list of empty",
			msg:  msg,
			typ:  "HoldsVerEmptyList",
			text: "(mylist = [(), ()])\n",
			data: []byte{
				0, 0, 0, 0, 3, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 1, 0,
				1, 0, 0, 0, 7, 0, 0, 0,
				8, 0, 0, 0, 0, 0, 0, 0,
			},
		})
	}

	return tests
}

func TestMarshalShouldMatchData(t *testing.T) {
	for _, test := range makeMarshalTests(t) {
		if test.data == nil {
			// TODO(light): backfill all data
			continue
		}
		data, err := test.msg.Marshal()
		if err != nil {
			t.Errorf("%s: marshal error: %v", test.name, err)
			continue
		}
		want, err := encodeTestMessage(test.typ, test.text, test.data)
		if err != nil {
			t.Errorf("%s: %v", test.name, err)
			continue
		}
		if !bytes.Equal(data, want) {
			t.Errorf("%s: Marshal returned:\n%s\nwant:\n%s", test.name, hex.Dump(data), hex.Dump(want))
		}
	}
}

func TestMarshalShouldMatchTextWhenDecoded(t *testing.T) {
	// TODO(light): skip test when tool not found
	for _, test := range makeMarshalTests(t) {
		data, err := test.msg.Marshal()
		if err != nil {
			t.Errorf("%s: marshal error: %v", test.name, err)
			continue
		}
		text := string(CapnpDecode(data, test.typ))
		if text != test.text {
			t.Errorf("%s: decoded to:\n%q; want:\n%q", test.name, text, test.text)
		}
	}
}

func TestMarshalPackedShouldMatchTextWhenDecoded(t *testing.T) {
	// TODO(light): skip test when tool not found
	for _, test := range makeMarshalTests(t) {
		data, err := test.msg.MarshalPacked()
		if err != nil {
			t.Errorf("%s: marshal error: %v", test.name, err)
			continue
		}
		text := CapnpDecodeBuf(data, "", "", test.typ, true)
		// TODO(light): don't trim
		if want := strings.TrimSpace(test.text); text != want {
			t.Errorf("%s: decoded to:\n%q; want:\n%q", test.name, text, want)
		}
	}
}

func TestZDataAccessors(t *testing.T) {
	data := mustEncodeTestMessage(t, "Z", `(zdata = (data = "\x00\x01\x02\x03\x04\x05\x06\a\b\t\n\v\f\r\x0e\x0f\x10\x11\x12\x13"))`, []byte{
		0, 0, 0, 0, 8, 0, 0, 0,
		0, 0, 0, 0, 2, 0, 1, 0,
		28, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 1, 0,
		1, 0, 0, 0, 162, 0, 0, 0,
		0, 1, 2, 3, 4, 5, 6, 7,
		8, 9, 10, 11, 12, 13, 14, 15,
		16, 17, 18, 19, 0, 0, 0, 0,
	})

	msg, err := capnp.Unmarshal(data)
	if err != nil {
		t.Fatal("Unmarshal:", err)
	}
	z, err := air.ReadRootZ(msg)
	if err != nil {
		t.Fatal("ReadRootZ:", err)
	}

	if z.Which() != air.Z_Which_zdata {
		t.Fatalf("z.Which() = %v; want zdata", z.Which())
	}
	zdata, err := z.Zdata()
	if err != nil {
		t.Fatal("z.Zdata():", err)
	}
	d, err := zdata.Data()
	if err != nil {
		t.Fatal("z.Zdata().Data():", err)
	}
	if len(d) != 20 {
		t.Errorf("z.Zdata().Data() len = %d; want 20", len(d))
	}
	for i := range d {
		if d[i] != byte(i) {
			t.Errorf("z.Zdata().Data()[%d] = %d; want %d", i, d[i], i)
		}
	}
}

func TestInterfaceSet(t *testing.T) {
	cl := air.Echo{Client: capnp.ErrorClient(errors.New("foo"))}
	_, s, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		t.Fatal(err)
	}
	base, err := air.NewRootEchoBase(s)
	if err != nil {
		t.Fatal(err)
	}

	base.SetEcho(cl)

	if base.Echo() != cl {
		t.Errorf("base.Echo() = %#v; want %#v", base.Echo(), cl)
	}
}

func TestInterfaceSetNull(t *testing.T) {
	cl := air.Echo{Client: capnp.ErrorClient(errors.New("foo"))}
	msg, s, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		t.Fatal(err)
	}
	base, err := air.NewRootEchoBase(s)
	if err != nil {
		t.Fatal(err)
	}
	base.SetEcho(cl)

	base.SetEcho(air.Echo{})

	if e := base.Echo().Client; e != nil {
		t.Errorf("base.Echo() = %#v; want nil", e)
	}
	if len(msg.CapTable) != 1 {
		t.Errorf("msg.CapTable = %#v; want len = 1", msg.CapTable)
	}
}

func TestInterfaceCopyToOtherMessage(t *testing.T) {
	cl := air.Echo{Client: capnp.ErrorClient(errors.New("foo"))}
	_, s1, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		t.Fatal(err)
	}
	base1, err := air.NewRootEchoBase(s1)
	if err != nil {
		t.Fatal(err)
	}
	if err := base1.SetEcho(cl); err != nil {
		t.Fatal(err)
	}

	_, s2, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		t.Fatal(err)
	}
	hoth2, err := air.NewRootHoth(s2)
	if err != nil {
		t.Fatal(err)
	}
	if err := hoth2.SetBase(base1); err != nil {
		t.Fatal(err)
	}

	if base, err := hoth2.Base(); err != nil {
		t.Errorf("hoth2.Base() error: %v", err)
	} else if base.Echo() != cl {
		t.Errorf("hoth2.Base().Echo() = %#v; want %#v", base.Echo(), cl)
	}
	tab2 := s2.Message().CapTable
	if len(tab2) == 1 {
		if tab2[0] != cl.Client {
			t.Errorf("s2.Message().CapTable[0] = %#v; want %#v", tab2[0], cl.Client)
		}
	} else {
		t.Errorf("len(s2.Message().CapTable) = %d; want 1", len(tab2))
	}
}

func TestInterfaceCopyToOtherMessageWithCaps(t *testing.T) {
	cl := air.Echo{Client: capnp.ErrorClient(errors.New("foo"))}
	_, s1, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		t.Fatal(err)
	}
	base1, err := air.NewRootEchoBase(s1)
	if err != nil {
		t.Fatal(err)
	}
	if err := base1.SetEcho(cl); err != nil {
		t.Fatal(err)
	}

	_, s2, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		t.Fatal(err)
	}
	s2.Message().AddCap(nil)
	hoth2, err := air.NewRootHoth(s2)
	if err != nil {
		t.Fatal(err)
	}
	if err := hoth2.SetBase(base1); err != nil {
		t.Fatal(err)
	}

	if base, err := hoth2.Base(); err != nil {
		t.Errorf("hoth2.Base() error: %v", err)
	} else if base.Echo() != cl {
		t.Errorf("hoth2.Base().Echo() = %#v; want %#v", base.Echo(), cl)
	}
	tab2 := s2.Message().CapTable
	if len(tab2) != 2 {
		t.Errorf("len(s2.Message().CapTable) = %d; want 2", len(tab2))
	}
}

// demonstrate and test serialization to List(List(Struct(List))), nested lists.

// start with smaller Struct(List)
func Test001StructList(t *testing.T) {

	cv.Convey("Given type Nester1 struct { Strs []string } in go, where Nester1 is a struct, and a mirror/parallel capnp struct air.Nester1Capn { strs @0: List(Text); } defined in the aircraftlib schema", t, func() {
		cv.Convey("When we Save() Nester to capn and then Load() it back, the data should match, so that we have working Struct(List) serialization and deserializatoin in go-capnproto", func() {

			// Does Nester1 alone serialization and deser okay?
			rw := Nester1{Strs: []string{"xenophilia", "watchowski"}}

			var o bytes.Buffer
			rw.Save(&o)

			msg, err := capnp.Unmarshal(o.Bytes())
			cv.So(err, cv.ShouldEqual, nil)
			seg, err := msg.Segment(0)
			cv.So(err, cv.ShouldEqual, nil)

			text := CapnpDecodeSegment(seg, "", schemaPath, "Nester1Capn")
			if false {
				fmt.Printf("text = '%s'\n", text)
			}
			rw2 := &Nester1{}
			rw2.Load(&o)

			//fmt.Printf("rw = '%#v'\n", rw)
			//fmt.Printf("rw2 = '%#v'\n", rw2)

			same := reflect.DeepEqual(&rw, rw2)
			cv.So(same, cv.ShouldEqual, true)
		})
	})
}

func Test002ListListStructList(t *testing.T) {

	cv.Convey("Given type RWTest struct { NestMatrix [][]Nester1; } in go, where Nester1 is a struct, and a mirror/parallel capnp struct air.RWTestCapn { nestMatrix @0: List(List(Nester1Capn)); } defined in the aircraftlib schema", t, func() {
		cv.Convey("When we Save() RWTest to capn and then Load() it back, the data should match, so that we have working List(List(Struct)) serialization and deserializatoin in go-capnproto", func() {

			// full RWTest
			rw := RWTest{
				NestMatrix: [][]Nester1{
					[]Nester1{
						Nester1{Strs: []string{"z", "w"}},
						Nester1{Strs: []string{"q", "r"}},
					},
					[]Nester1{
						Nester1{Strs: []string{"zebra", "wally"}},
						Nester1{Strs: []string{"qubert", "rocks"}},
					},
				},
			}

			var o bytes.Buffer
			rw.Save(&o)

			msg, err := capnp.Unmarshal(o.Bytes())
			cv.So(err, cv.ShouldEqual, nil)
			seg, err := msg.Segment(0)
			cv.So(err, cv.ShouldEqual, nil)

			text := CapnpDecodeSegment(seg, "", schemaPath, "RWTestCapn")

			if false {
				fmt.Printf("text = '%s'\n", text)
			}

			rw2 := &RWTest{}
			rw2.Load(&o)

			//fmt.Printf("rw = '%#v'\n", rw)
			//fmt.Printf("rw2 = '%#v'\n", rw2)

			same := reflect.DeepEqual(&rw, rw2)
			cv.So(same, cv.ShouldEqual, true)
		})
	})
}

type Nester1 struct {
	Strs []string
}

type RWTest struct {
	NestMatrix [][]Nester1
}

func (s *Nester1) Save(w io.Writer) {
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		panic(err)
	}
	msg.SetRoot(Nester1GoToCapn(seg, s))
	data, err := msg.Marshal()
	if err != nil {
		panic(err)
	}
	w.Write(data)
}

func Nester1GoToCapn(seg *capnp.Segment, src *Nester1) air.Nester1Capn {
	//fmt.Printf("\n\n   Nester1GoToCapn sees seg = '%#v'\n", seg)
	dest, _ := air.NewNester1Capn(seg)

	mylist1, _ := capnp.NewTextList(seg, int32(len(src.Strs)))
	for i := range src.Strs {
		mylist1.Set(i, string(src.Strs[i]))
	}
	dest.SetStrs(mylist1)

	//fmt.Printf("after Nester1GoToCapn setting\n")
	return dest
}

func Nester1CapnToGo(src air.Nester1Capn, dest *Nester1) *Nester1 {
	if dest == nil {
		dest = &Nester1{}
	}
	srcStrs, _ := src.Strs()
	dest.Strs = make([]string, srcStrs.Len())
	for i := range dest.Strs {
		dest.Strs[i], _ = srcStrs.At(i)
	}

	return dest
}

func (s *Nester1) Load(r io.Reader) {
	capMsg, err := capnp.NewDecoder(r).Decode()
	if err != nil {
		panic(fmt.Errorf("capnp.ReadFromStream error: %s", err))
	}
	z, _ := air.ReadRootNester1Capn(capMsg)
	Nester1CapnToGo(z, s)
}

func (s *RWTest) Save(w io.Writer) {
	msg, seg, _ := capnp.NewMessage(capnp.SingleSegment(nil))
	msg.SetRoot(RWTestGoToCapn(seg, s))
	data, _ := msg.Marshal()
	w.Write(data)
}

func (s *RWTest) Load(r io.Reader) {
	capMsg, err := capnp.NewDecoder(r).Decode()
	if err != nil {
		panic(fmt.Errorf("capnp.ReadFromStream error: %s", err))
	}
	z, _ := air.ReadRootRWTestCapn(capMsg)
	RWTestCapnToGo(z, s)
}

func RWTestCapnToGo(src air.RWTestCapn, dest *RWTest) *RWTest {
	if dest == nil {
		dest = &RWTest{}
	}
	var n int
	srcMatrix, _ := src.NestMatrix()
	// NestMatrix
	n = srcMatrix.Len()
	dest.NestMatrix = make([][]Nester1, n)
	for i := 0; i < n; i++ {
		sm, _ := srcMatrix.At(i)
		dest.NestMatrix[i] = Nester1CapnListToSliceNester1(air.Nester1Capn_List{List: capnp.ToList(sm)})
	}

	return dest
}

func RWTestGoToCapn(seg *capnp.Segment, src *RWTest) air.RWTestCapn {
	dest, err := air.NewRWTestCapn(seg)
	if err != nil {
		panic(err)
	}

	// NestMatrix -> Nester1Capn (go slice to capn list)
	if len(src.NestMatrix) > 0 {
		plist, err := capnp.NewPointerList(seg, int32(len(src.NestMatrix)))
		if err != nil {
			panic(err)
		}
		for i, ele := range src.NestMatrix {
			err := plist.Set(i, SliceNester1ToNester1CapnList(seg, ele))
			if err != nil {
				panic(err)
			}
		}
		dest.SetNestMatrix(plist)
	}

	return dest
}

func Nester1CapnListToSliceNester1(p air.Nester1Capn_List) []Nester1 {
	v := make([]Nester1, p.Len())
	for i := range v {
		Nester1CapnToGo(p.At(i), &v[i])
	}
	return v
}

func SliceNester1ToNester1CapnList(seg *capnp.Segment, m []Nester1) air.Nester1Capn_List {
	lst, err := air.NewNester1Capn_List(seg, int32(len(m)))
	if err != nil {
		panic(err)
	}
	for i := range m {
		err := lst.Set(i, Nester1GoToCapn(seg, &m[i]))
		if err != nil {
			panic(err)
		}
	}
	return lst
}

func SliceStringToTextList(seg *capnp.Segment, m []string) capnp.TextList {
	lst, err := capnp.NewTextList(seg, int32(len(m)))
	if err != nil {
		panic(err)
	}
	for i := range m {
		lst.Set(i, string(m[i]))
	}
	return lst
}

func TextListToSliceString(p capnp.TextList) []string {
	v := make([]string, p.Len())
	for i := range v {
		s, err := p.At(i)
		if err != nil {
			panic(err)
		}
		v[i] = s
	}
	return v
}

func TestDataVersioningAvoidsUnnecessaryTruncation(t *testing.T) {
	in := mustEncodeTestMessage(t, "VerTwoDataTwoPtr", "(val = 9, duo = 8, ptr1 = (val = 77), ptr2 = (val = 55))", []byte{
		0, 0, 0, 0, 7, 0, 0, 0,
		0, 0, 0, 0, 2, 0, 2, 0,
		9, 0, 0, 0, 0, 0, 0, 0,
		8, 0, 0, 0, 0, 0, 0, 0,
		4, 0, 0, 0, 1, 0, 0, 0,
		4, 0, 0, 0, 1, 0, 0, 0,
		77, 0, 0, 0, 0, 0, 0, 0,
		55, 0, 0, 0, 0, 0, 0, 0,
	})
	want := mustEncodeTestMessage(t, "Wrap2x2", "(mightNotBeReallyEmpty = (val = 9, duo = 8, ptr1 = (val = 77), ptr2 = (val = 55)))", []byte{
		0, 0, 0, 0, 8, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 1, 0,
		0, 0, 0, 0, 2, 0, 2, 0,
		9, 0, 0, 0, 0, 0, 0, 0,
		8, 0, 0, 0, 0, 0, 0, 0,
		4, 0, 0, 0, 1, 0, 0, 0,
		4, 0, 0, 0, 1, 0, 0, 0,
		77, 0, 0, 0, 0, 0, 0, 0,
		55, 0, 0, 0, 0, 0, 0, 0,
	})

	msg, err := capnp.Unmarshal(in)
	if err != nil {
		t.Fatal("Unmarshal:", err)
	}

	// Read in the message as if it's an old client (less fields in schema).
	oldRoot, err := air.ReadRootVerEmpty(msg)
	if err != nil {
		t.Fatal("ReadRootVerEmpty:", err)
	}

	// Store the larger message into another segment.
	freshMsg, freshSeg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		t.Fatal("NewMessage:", err)
	}
	wrapEmpty, err := air.NewRootWrapEmpty(freshSeg)
	if err != nil {
		t.Fatal("NewRootWrapEmpty:", err)
	}
	if err := wrapEmpty.SetMightNotBeReallyEmpty(oldRoot); err != nil {
		t.Fatal("SetMightNotBeReallyEmpty:", err)
	}

	// Verify that it matches the expected serialization.
	out, err := freshMsg.Marshal()
	if err != nil {
		t.Fatal("Marshal:", err)
	}
	if !bytes.Equal(out, want) {
		t.Errorf("After copy, data is:\n%s\nwant:\n%s", hex.Dump(out), hex.Dump(want))
	}
}

func TestZserverAccessors(t *testing.T) {
	in := mustEncodeTestMessage(t, "Zserver", `(waitingjobs = [(cmd = "abc"), (cmd = "xyz")])`, []byte{
		0, 0, 0, 0, 9, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 1, 0,
		1, 0, 0, 0, 39, 0, 0, 0,
		8, 0, 0, 0, 0, 0, 2, 0,
		13, 0, 0, 0, 34, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		9, 0, 0, 0, 34, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		97, 98, 99, 0, 0, 0, 0, 0,
		120, 121, 122, 0, 0, 0, 0, 0,
	})

	msg, err := capnp.Unmarshal(in)
	if err != nil {
		t.Fatal("Unmarshal:", err)
	}

	zserver, err := air.ReadRootZserver(msg)
	if err != nil {
		t.Fatal("ReadRootZserver:", err)
	}
	joblist, err := zserver.Waitingjobs()
	if err != nil {
		t.Fatal("Zserver.waitingjobs:", err)
	}
	if joblist.Len() != 2 {
		t.Fatalf("len(Zserver.waitingjobs) = %d; want 2", joblist.Len())
	}
	checkCmd := func(i int, want string) {
		cmd, err := joblist.At(i).Cmd()
		if err != nil {
			t.Errorf("Zserver.waitingjobs[%d].cmd error: %v", i, err)
			return
		}
		if cmd != want {
			t.Errorf("Zserver.waitingjobs[%d].cmd = %q; want %q", i, cmd, want)
		}
	}
	checkCmd(0, "abc")
	checkCmd(1, "xyz")
}

func TestEnumFromString(t *testing.T) {
	tests := []struct {
		s  string
		ap air.Airport
	}{
		{"jfk", air.Airport_jfk},
		{"notEverMatching", 0},
	}
	for _, test := range tests {
		if ap := air.AirportFromString(test.s); ap != test.ap {
			t.Errorf("air.AirportFromString(%q) = %v; want %v", test.s, ap, test.ap)
		}
	}
}

func ShowSeg(msg string, seg *capnp.Segment) []byte {
	b, err := seg.Message().Marshal()
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", msg)
	ShowBytes(b, 10)
	return b
}

func TestDefaultStructField(t *testing.T) {
	_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		t.Fatal(err)
	}
	root, err := air.NewRootStackingRoot(seg)
	if err != nil {
		t.Fatal(err)
	}

	a, err := root.AWithDefault()

	if err != nil {
		t.Error("StackingRoot.aWithDefault error:", err)
	}
	if a.Num() != 42 {
		t.Errorf("StackingRoot.aWithDefault = %d; want 42", a.Num())
	}
}

func TestDataTextCopyOptimization(t *testing.T) {
	_, seg1, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		t.Fatal(err)
	}
	root, err := air.NewRootNester1Capn(seg1)
	if err != nil {
		t.Fatal(err)
	}
	_, seg2, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		t.Fatal(err)
	}
	strsl, err := capnp.NewTextList(seg2, 256)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < strsl.Len(); i++ {
		strsl.Set(i, "testess")
	}

	err = root.SetStrs(strsl)

	if err != nil {
		t.Fatal(err)
	}
	strsl, err = root.Strs()
	if err != nil {
		t.Fatal(err)
	}
	if strsl.Len() != 256 {
		t.Errorf("strsl.Len() = %d; want 256", strsl.Len())
	}
	for i := 0; i < strsl.Len(); i++ {
		s, err := strsl.At(i)
		if err != nil {
			t.Errorf("strsl.At(%d) error: %v", i, err)
			continue
		}
		if s != "testess" {
			t.Errorf("strsl.At(%d) = %q; want \"testess\"", i, s)
		}
	}
}

// highlight how much faster text movement between segments
// is when special casing Text and Data
//
// run this test with capnp.go:1334-1341 commented in/out to compare.
//
func BenchmarkTextMovementBetweenSegments(b *testing.B) {

	buf := make([]byte, 1<<21)
	buf2 := make([]byte, 1<<21)

	text := make([]byte, 1<<20)
	for i := range text {
		text[i] = byte(65 + rand.Int()%26)
	}
	//stext := string(text)
	//fmt.Printf("text = %#v\n", stext)

	astr := make([]string, 1000)
	for i := range astr {
		astr[i] = string(text[i*1000 : (i+1)*1000])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, seg, _ := capnp.NewMessage(capnp.SingleSegment(buf[:0]))
		_, scratch, _ := capnp.NewMessage(capnp.SingleSegment(buf2[:0]))

		ht, _ := air.NewRootHoldsText(seg)
		tl, _ := capnp.NewTextList(scratch, 1000)

		for j := 0; j < 1000; j++ {
			tl.Set(j, astr[j])
		}

		ht.SetLst(tl)

	}
}

func TestV1DataVersioningBiggerToEmpty(t *testing.T) {
	in := mustEncodeTestMessage(t, "HoldsVerTwoDataList", "(mylist = [(val = 27, duo = 26), (val = 42, duo = 41)])", []byte{
		0, 0, 0, 0, 7, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 1, 0,
		1, 0, 0, 0, 39, 0, 0, 0,
		8, 0, 0, 0, 2, 0, 0, 0,
		27, 0, 0, 0, 0, 0, 0, 0,
		26, 0, 0, 0, 0, 0, 0, 0,
		42, 0, 0, 0, 0, 0, 0, 0,
		41, 0, 0, 0, 0, 0, 0, 0,
	})

	remsg, err := capnp.Unmarshal(in)
	if err != nil {
		t.Fatal("Unmarshal:", err)
	}

	// 0 data
	func() {
		reHolder0, err := air.ReadRootHoldsVerEmptyList(remsg)
		if err != nil {
			t.Error("ReadRootHoldsVerEmptyList:", err)
			return
		}
		list0, err := reHolder0.Mylist()
		if err != nil {
			t.Error("HoldsVerEmptyList.mylist:", err)
			return
		}
		if list0.Len() != 2 {
			t.Errorf("len(HoldsVerEmptyList.mylist) = %d; want 2", list0.Len())
		}
	}()

	// 1 datum
	func() {
		reHolder1, err := air.ReadRootHoldsVerOneDataList(remsg)
		if err != nil {
			t.Error("ReadRootHoldsVerOneDataList:", err)
			return
		}
		list1, err := reHolder1.Mylist()
		if err != nil {
			t.Error("HoldsVerOneDataList.mylist:", err)
			return
		}
		if list1.Len() == 2 {
			if v := list1.At(0).Val(); v != 27 {
				t.Errorf("HoldsVerOneDataList.mylist[0].val = %d; want 27", v)
			}
			if v := list1.At(1).Val(); v != 42 {
				t.Errorf("HoldsVerOneDataList.mylist[1].val = %d; want 42", v)
			}
		} else {
			t.Errorf("len(HoldsVerOneDataList.mylist) = %d; want 2", list1.Len())
		}
	}()

	// 2 data
	func() {
		reHolder2, err := air.ReadRootHoldsVerTwoDataList(remsg)
		if err != nil {
			t.Error("ReadRootHoldsVerTwoDataList:", err)
			return
		}
		list2, err := reHolder2.Mylist()
		if err != nil {
			t.Error("HoldsVerTwoDataList.mylist:", err)
			return
		}
		if list2.Len() == 2 {
			if v := list2.At(0).Val(); v != 27 {
				t.Errorf("HoldsVerTwoDataList.mylist[0].val = %d; want 27", v)
			}
			if v := list2.At(0).Duo(); v != 26 {
				t.Errorf("HoldsVerTwoDataList.mylist[0].duo = %d; want 26", v)
			}
			if v := list2.At(1).Val(); v != 42 {
				t.Errorf("HoldsVerTwoDataList.mylist[1].val = %d; want 42", v)
			}
			if v := list2.At(1).Duo(); v != 41 {
				t.Errorf("HoldsVerTwoDataList.mylist[1].duo = %d; want 41", v)
			}
		} else {
			t.Errorf("len(HoldsVerTwoDataList.mylist) = %d; want 2", list2.Len())
		}
	}()
}

func TestV1DataVersioningEmptyToBigger(t *testing.T) {
	in := mustEncodeTestMessage(t, "HoldsVerEmptyList", "(mylist = [(),()])", []byte{
		0, 0, 0, 0, 3, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 1, 0,
		1, 0, 0, 0, 7, 0, 0, 0,
		8, 0, 0, 0, 0, 0, 0, 0,
	})

	remsg, err := capnp.Unmarshal(in)
	if err != nil {
		t.Fatal("Unmarshal:", err)
	}

	reHolder1, err := air.ReadRootHoldsVerOneDataList(remsg)
	if err != nil {
		t.Fatal("ReadRootHoldsVerOneDataList:", err)
	}
	list1, err := reHolder1.Mylist()
	if err != nil {
		t.Fatal("HoldsVerOneDataList.mylist:", err)
	}
	if list1.Len() == 2 {
		if v := list1.At(0).Val(); v != 0 {
			t.Errorf("HoldsVerOneDataList.mylist[0].val = %d; want 0", v)
		}
		if v := list1.At(1).Val(); v != 0 {
			t.Errorf("HoldsVerOneDataList.mylist[1].val = %d; want 0", v)
		}
	} else {
		t.Errorf("len(HoldsVerOneDataList.mylist) = %d; want 2", list1.Len())
	}

	reHolder2, err := air.ReadRootHoldsVerTwoDataList(remsg)
	if err != nil {
		t.Fatal("ReadRootHoldsVerOneDataList:", err)
	}
	list2, err := reHolder2.Mylist()
	if err != nil {
		t.Fatal("HoldsVerOneDataList.mylist:", err)
	}
	if list2.Len() == 2 {
		if v := list2.At(0).Val(); v != 0 {
			t.Errorf("HoldsVerTwoDataList.mylist[0].val = %d; want 0", v)
		}
		if v := list2.At(0).Duo(); v != 0 {
			t.Errorf("HoldsVerTwoDataList.mylist[0].duo = %d; want 0", v)
		}
		if v := list2.At(1).Val(); v != 0 {
			t.Errorf("HoldsVerTwoDataList.mylist[1].val = %d; want 0", v)
		}
		if v := list2.At(1).Duo(); v != 0 {
			t.Errorf("HoldsVerTwoDataList.mylist[1].duo = %d; want 0", v)
		}
	} else {
		t.Errorf("len(HoldsVerTwoDataList.mylist) = %d; want 2", list2.Len())
	}
}

func TestDataVersioningZeroPointersToMore(t *testing.T) {
	in := mustEncodeTestMessage(t, "HoldsVerEmptyList", "(mylist = [(),()])", []byte{
		0, 0, 0, 0, 3, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 1, 0,
		1, 0, 0, 0, 7, 0, 0, 0,
		8, 0, 0, 0, 0, 0, 0, 0,
	})

	remsg, err := capnp.Unmarshal(in)
	if err != nil {
		t.Fatal("Unmarshal:", err)
	}
	reHolder, err := air.ReadRootHoldsVerTwoTwoList(remsg)
	if err != nil {
		t.Fatal("ReadRootHoldsVerTwoTwoList:", err)
	}
	list22, err := reHolder.Mylist()
	if err != nil {
		t.Fatal("HoldsVerTwoTwoList.mylist:", err)
	}
	if list22.Len() != 2 {
		t.Errorf("len(HoldsVerTwoTwoList.mylist) = %d; want 2", list22.Len())
	}
	for i := 0; i < list22.Len(); i++ {
		ele := list22.At(i)
		if val := ele.Val(); val != 0 {
			t.Errorf("HoldsVerTwoTwoList.mylist[%d].val = %d; want 0", i, val)
		}
		if duo := ele.Duo(); duo != 0 {
			t.Errorf("HoldsVerTwoTwoList.mylist[%d].duo = %d; want 0", i, duo)
		}
		if ptr1, err := ele.Ptr1(); err != nil {
			t.Errorf("HoldsVerTwoTwoList.mylist[%d].ptr1: %v", i, err)
		} else if capnp.IsValid(ptr1) {
			t.Errorf("HoldsVerTwoTwoList.mylist[%d].ptr1 = %#v; want invalid (nil)", i, ptr1)
		}
		if ptr2, err := ele.Ptr2(); err != nil {
			t.Errorf("HoldsVerTwoTwoList.mylist[%d].ptr2: %v", i, err)
		} else if capnp.IsValid(ptr2) {
			t.Errorf("HoldsVerTwoTwoList.mylist[%d].ptr2 = %#v; want invalid (nil)", i, ptr2)
		}
	}
}

func TestDataVersioningZeroPointersToTwo(t *testing.T) {
	in := mustEncodeTestMessage(
		t,
		"HoldsVerTwoTwoList",
		`(mylist = [
			(val = 27, duo = 26, ptr1 = (val = 25), ptr2 = (val = 23)),
			(val = 42, duo = 41, ptr1 = (val = 40), ptr2 = (val = 38))])`,
		[]byte{
			0, 0, 0, 0, 15, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 1, 0,
			1, 0, 0, 0, 71, 0, 0, 0,
			8, 0, 0, 0, 2, 0, 2, 0,
			27, 0, 0, 0, 0, 0, 0, 0,
			26, 0, 0, 0, 0, 0, 0, 0,
			20, 0, 0, 0, 1, 0, 0, 0,
			20, 0, 0, 0, 1, 0, 0, 0,
			42, 0, 0, 0, 0, 0, 0, 0,
			41, 0, 0, 0, 0, 0, 0, 0,
			12, 0, 0, 0, 1, 0, 0, 0,
			12, 0, 0, 0, 1, 0, 0, 0,
			25, 0, 0, 0, 0, 0, 0, 0,
			23, 0, 0, 0, 0, 0, 0, 0,
			40, 0, 0, 0, 0, 0, 0, 0,
			38, 0, 0, 0, 0, 0, 0, 0,
		})

	remsg, err := capnp.Unmarshal(in)
	if err != nil {
		t.Fatal("Unmarshal:", err)
	}

	// 0 pointers
	func() {
		reHolder, err := air.ReadRootHoldsVerEmptyList(remsg)
		if err != nil {
			t.Error("ReadRootHoldsVerEmptyList:", err)
			return
		}
		list, err := reHolder.Mylist()
		if err != nil {
			t.Error("HoldsVerEmptyList.mylist:", err)
			return
		}
		if list.Len() != 2 {
			t.Errorf("len(HoldsVerEmptyList.mylist) = %d; want 2", list.Len())
		}
	}()

	// 1 pointer
	func() {
		holder, err := air.ReadRootHoldsVerOnePtrList(remsg)
		if err != nil {
			t.Error("ReadRootHoldsVerOnePtrList:", err)
			return
		}
		list, err := holder.Mylist()
		if err != nil {
			t.Error("HoldsVerOnePtrList.mylist:", err)
			return
		}
		if list.Len() != 2 {
			t.Errorf("len(HoldsVerOnePtrList.mylist) = %d; want 2", list.Len())
			return
		}
		check := func(i int, val int16) {
			p, err := list.At(i).Ptr()
			if err != nil {
				t.Errorf("HoldsVerOnePtrList.mylist[%d].ptr: %v", i, err)
				return
			}
			if p.Val() != val {
				t.Errorf("HoldsVerOnePtrList.mylist[%d].ptr.val = %d; want %d", i, p.Val(), val)
			}
		}
		check(0, 25)
		check(1, 40)
	}()

	// 2 pointers
	func() {
		holder, err := air.ReadRootHoldsVerTwoTwoPlus(remsg)
		if err != nil {
			t.Error("ReadRootHoldsVerTwoTwoPlus:", err)
			return
		}
		list, err := holder.Mylist()
		if err != nil {
			t.Error("HoldsVerTwoTwoPlus.mylist:", err)
			return
		}
		if list.Len() != 2 {
			t.Errorf("len(HoldsVerTwoTwoPlus.mylist) = %d; want 2", list.Len())
			return
		}
		check := func(i int, val1, val2 int16) {
			if p, err := list.At(i).Ptr1(); err != nil {
				t.Errorf("HoldsVerTwoTwoPlus.mylist[%d].ptr1: %v", i, err)
			} else if p.Val() != val1 {
				t.Errorf("HoldsVerTwoTwoPlus.mylist[%d].ptr1.val = %d; want %d", i, p.Val(), val1)
			}
			if p, err := list.At(i).Ptr2(); err != nil {
				t.Errorf("HoldsVerTwoTwoPlus.mylist[%d].ptr2: %v", i, err)
			} else if p.Val() != val2 {
				t.Errorf("HoldsVerTwoTwoPlus.mylist[%d].ptr2.val = %d; want %d", i, p.Val(), val2)
			}
		}
		check(0, 25, 23)
		check(1, 40, 38)
	}()
}

func TestVoidUnionSetters(t *testing.T) {
	want := mustEncodeTestMessage(t, "VoidUnion", "(b = void)", []byte{
		0, 0, 0, 0, 2, 0, 0, 0,
		0, 0, 0, 0, 1, 0, 0, 0,
		1, 0, 0, 0, 0, 0, 0, 0,
	})

	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		t.Fatal(err)
	}
	voidUnion, err := air.NewRootVoidUnion(seg)
	if err != nil {
		t.Fatal(err)
	}
	voidUnion.SetB()

	act, err := msg.Marshal()
	if err != nil {
		t.Fatal("msg.Marshal():", err)
	}
	if !bytes.Equal(act, want) {
		t.Errorf("msg.Marshal() =\n%s\n; want:\n%s", hex.Dump(act), hex.Dump(want))
	}
}
