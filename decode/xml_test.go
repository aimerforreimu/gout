package decode

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewXMLDecode(t *testing.T) {
	x := NewXMLDecode(nil)
	assert.Nil(t, x)
}

type xmlTest struct {
	r    *bytes.Buffer
	need interface{}
	got  interface{}
}

func testDecodeXML(t *testing.T, funcName string) {
	type xmlVal struct {
		A string `xml:"a"`
		B string `xml:"b"`
	}

	tests := []xmlTest{
		{r: bytes.NewBufferString(`<xmlVal><a>a</a><b>b</b></xmlVal>`), need: &xmlVal{A: "a", B: "b"}, got: &xmlVal{}},
		{r: bytes.NewBufferString(`<xmlVal><a>aaa</a><b>bbb</b></xmlVal>`), need: &xmlVal{A: "aaa", B: "bbb"}, got: &xmlVal{}},
	}

	for _, v := range tests {
		if funcName == "TestDecode" {
			x := NewXMLDecode(v.got)
			x.Decode(v.r)
		} else {
			DecodeXML(v.r, v.got)
		}
		assert.Equal(t, v.need, v.got)
	}
}

func Test_xml_DecodeXML(t *testing.T) {
	testDecodeXML(t, "TestDecodeXML")
}

func Test_xml_Decode(t *testing.T) {
	testDecodeXML(t, "TestDecode")
}
