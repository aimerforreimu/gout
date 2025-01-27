package gout

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/guonaihong/gout/color"
	"github.com/guonaihong/gout/core"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

type testDataFlow struct {
	send  bool
	total int32
}

type data struct {
	Id   int    `json:"id" xml:"id"`
	Data string `json:"data" xml:"data"`
}

type BindTest struct {
	InBody   interface{}
	OutBody  interface{}
	httpCode int
}

func TestBindXML(t *testing.T) {
	var d, d2 data
	router := func() *gin.Engine {
		router := gin.Default()

		router.POST("/test.xml", func(c *gin.Context) {
			var d3 data
			err := c.BindXML(&d3)
			assert.NoError(t, err)
			c.XML(200, d3)
		})
		return router
	}()

	ts := httptest.NewServer(http.HandlerFunc(router.ServeHTTP))
	defer ts.Close()

	g := New(nil)

	d.Id = 3
	d.Data = "test data"

	code := 200

	err := g.POST(ts.URL + "/test.xml").SetXML(&d).BindXML(&d2).Code(&code).Do()

	assert.NoError(t, err)
	assert.Equal(t, code, 200)
	assert.Equal(t, d, d2)
}

func TestBindYAML(t *testing.T) {
	var d, d2 data
	router := func() *gin.Engine {
		router := gin.Default()

		router.POST("/test.yaml", func(c *gin.Context) {
			var d3 data
			err := c.BindYAML(&d3)
			assert.NoError(t, err)
			c.YAML(200, d3)
		})
		return router
	}()

	ts := httptest.NewServer(http.HandlerFunc(router.ServeHTTP))
	defer ts.Close()

	g := New(nil)

	d.Id = 3
	d.Data = "test yaml data"

	code := 200

	err := g.POST(ts.URL + "/test.yaml").SetYAML(&d).BindYAML(&d2).Code(&code).Do()

	assert.NoError(t, err)
	assert.Equal(t, code, 200)
	assert.Equal(t, d, d2)
}

func TestBindJSON(t *testing.T) {
	var d3 data
	router := func() *gin.Engine {
		router := gin.Default()

		router.POST("/test.json", func(c *gin.Context) {
			c.BindJSON(&d3)
			c.JSON(200, d3)
		})

		return router
	}()

	ts := httptest.NewServer(http.HandlerFunc(router.ServeHTTP))
	defer ts.Close()

	tests := []BindTest{
		{InBody: data{Id: 9, Data: "早上测试结构体"}, OutBody: data{}},
		{InBody: H{"id": 10, "data": "早上测试map"}, OutBody: data{}},
	}

	for k, _ := range tests {
		t.Logf("outbody type:%T:%p\n", tests[k].OutBody, &tests[k].OutBody)

		err := POST(ts.URL + "/test.json").
			SetJSON(&tests[k].InBody).
			BindJSON(&tests[k].OutBody).
			Code(&tests[k].httpCode).
			Do()
		if err != nil {
			t.Errorf("send fail:%s\n", err)
		}
		assert.NoError(t, err)

		assert.Equal(t, tests[k].httpCode, 200)

		if tests[k].OutBody.(map[string]interface{})["id"].(float64) != float64(d3.Id) {
			t.Errorf("got:%#v(P:%p), want:%#v(T:%T)\n", tests[k].OutBody, &tests[k].OutBody, tests[k].InBody, tests[k].InBody)
		}

		/*
			if !reflect.DeepEqual(&tests[k].InBody, &tests[k].OutBody) {
				t.Errorf("got:%#v(P:%p), want:%#v(T:%T)\n", tests[k].OutBody, &tests[k].OutBody, tests[k].InBody, tests[k].InBody)
			}
		*/

	}
}

func TestBindHeader(t *testing.T) {
	router := func() *gin.Engine {
		router := gin.Default()

		router.GET("/test.header", func(c *gin.Context) {
			c.Writer.Header().Add("sid", "sid-ok")
		})

		return router
	}()

	ts := httptest.NewServer(http.HandlerFunc(router.ServeHTTP))

	g := New(nil)

	type testHeader struct {
		Sid  string `header:"sid"`
		Code int
	}

	var tHeader testHeader
	err := g.GET(ts.URL + "/test.header").BindHeader(&tHeader).Code(&tHeader.Code).Do()
	assert.NoError(t, err)
	assert.Equal(t, tHeader.Code, 200)
	assert.Equal(t, tHeader.Sid, "sid-ok")
}

type testForm struct {
	Mode    string  `form:"mode"`
	Text    string  `form:"text"`
	Int     int     `form:"int"`
	Int8    int8    `form:"int8"`
	Int16   int16   `form:"int16"`
	Int32   int32   `form:"int32"`
	Int64   int64   `form:"int64"`
	Uint    uint    `form:"uint"`
	Uint8   uint8   `form:"uint8"`
	Uint16  uint16  `form:"uint16"`
	Uint32  uint32  `form:"uint32"`
	Uint64  uint64  `form:"uint64"`
	Float32 float32 `form:"float32"`
	Float64 float64 `form:"float64"`
	//Voice   []byte  `form-mem:"true"`  //测试从内存中构造
	//Voice2  []byte  `form-file:"true"` //测试从文件中读取
}

func setupForm(t *testing.T, reqTestForm testForm) *gin.Engine {
	router := gin.Default()
	router.POST("/test.form", func(c *gin.Context) {

		t2 := testForm{}
		err := c.Bind(&t2)
		assert.NoError(t, err)
		assert.Equal(t, reqTestForm, t2)
		/*
			assert.Equal(t, reqTestForm.Mode, t2.Mode)
			assert.Equal(t, reqTestForm.Text, t2.Text)
		*/
	})
	return router
}

type testForm2 struct {
	Mode    string  `form:"mode"`
	Text    string  `form:"text"`
	Int     int     `form:"int"`
	Uint    uint    `form:"uint"`
	Float32 float32 `form:"float32"`
	Float64 float64 `form:"float64"`

	Voice     *multipart.FileHeader `form:"voice"`
	Voice2    *multipart.FileHeader `form:"voice2"`
	ReqVoice  []byte
	ReqVoice2 []byte
}

func setupForm2(t *testing.T, reqTestForm testForm2) *gin.Engine {
	router := gin.Default()
	router.POST("/test.form", func(c *gin.Context) {

		t2 := testForm2{}
		err := c.Bind(&t2)
		assert.NoError(t, err)
		//assert.Equal(t, reqTestForm, t2)
		assert.Equal(t, reqTestForm.Mode, t2.Mode)
		assert.Equal(t, reqTestForm.Text, t2.Text)
		assert.Equal(t, reqTestForm.Int, t2.Int)
		assert.Equal(t, reqTestForm.Uint, t2.Uint)
		assert.Equal(t, reqTestForm.Float32, t2.Float32)
		assert.Equal(t, reqTestForm.Float64, t2.Float64)

		assert.NotNil(t, t2.Voice)
		fd, err := t2.Voice.Open()
		assert.NoError(t, err)
		defer fd.Close()

		all, err := ioutil.ReadAll(fd)
		assert.NoError(t, err)

		assert.Equal(t, reqTestForm.ReqVoice, all)
		//=============

		assert.NotNil(t, t2.Voice2)
		fd2, err := t2.Voice2.Open()
		assert.NoError(t, err)
		defer fd2.Close()

		all2, err := ioutil.ReadAll(fd2)
		assert.NoError(t, err)
		assert.Equal(t, reqTestForm.ReqVoice2, all2)
	})

	return router
}

func TestSetFormMap(t *testing.T) {
	reqTestForm := testForm2{
		Mode:      "A",
		Text:      "good morning",
		ReqVoice2: []byte("pcm pcm"),
		Int:       1,
		Uint:      2,
		Float32:   1.12,
		Float64:   3.14,
	}

	var err error
	reqTestForm.ReqVoice, err = ioutil.ReadFile("testdata/voice.pcm")
	assert.NoError(t, err)

	router := setupForm2(t, reqTestForm)

	ts := httptest.NewServer(http.HandlerFunc(router.ServeHTTP))
	defer ts.Close()

	g := New(nil)
	code := 0
	err = g.POST(ts.URL + "/test.form").
		SetForm(H{"mode": "A",
			"text":    "good morning",
			"voice":   core.FormFile("testdata/voice.pcm"),
			"voice2":  core.FormMem("pcm pcm"),
			"int":     1,
			"uint":    2,
			"float32": 1.12,
			"float64": 3.14,
		}).
		Code(&code).
		Do()

	assert.NoError(t, err)
}

func TestSetFormStruct(t *testing.T) {
	reqTestForm := testForm{
		Mode:    "A",
		Text:    "good morning",
		Int:     1,
		Int8:    2,
		Int16:   3,
		Int32:   4,
		Int64:   5,
		Uint:    6,
		Uint8:   7,
		Uint16:  8,
		Uint32:  9,
		Uint64:  10,
		Float32: 1.23,
		Float64: 3.14,
		//Voice: []byte("pcm data"),
	}

	router := setupForm(t, reqTestForm)

	ts := httptest.NewServer(http.HandlerFunc(router.ServeHTTP))
	defer ts.Close()

	g := New(nil)
	code := 0
	err := g.POST(ts.URL + "/test.form").
		//Debug(true).
		SetForm(&reqTestForm).
		Code(&code).
		Do()

	assert.NoError(t, err)
}

func TestSetHeaderMap(t *testing.T) {
}

func TestSetHeaderStruct(t *testing.T) {
	type testHeader2 struct {
		Q8 uint8 `header:"h8"`
	}

	type testHeader struct {
		Q1 string    `header:"h1"`
		Q2 int       `header:"h2"`
		Q3 float32   `header:"h3"`
		Q4 float64   `header:"h4"`
		Q5 time.Time `header:"h5" time_format:"unix"`
		Q6 time.Time `header:"h6" time_format:"unixNano"`
		Q7 time.Time `header:"h7" time_format:"2006-01-02"`
		testHeader2
	}

	h := testHeader{
		Q1: "v1",
		Q2: 2,
		Q3: 3.14,
		Q4: 3.1415,
		Q5: time.Date(2019, 7, 28, 14, 36, 0, 0, time.Local),
		Q6: time.Date(2019, 7, 28, 14, 36, 0, 1000, time.Local),
		Q7: time.Date(2019, 7, 28, 0, 0, 0, 0, time.Local),
		testHeader2: testHeader2{
			Q8: 8,
		},
	}

	router := func() *gin.Engine {
		router := gin.Default()
		router.GET("/test.header", func(c *gin.Context) {
			h2 := testHeader{}
			err := c.BindHeader(&h2)
			assert.NoError(t, err)

			assert.Equal(t, h, h2)
		})

		return router
	}()

	ts := httptest.NewServer(http.HandlerFunc(router.ServeHTTP))
	defer ts.Close()

	g := New(nil)
	code := 0

	err := g.GET(ts.URL + "/test.header").SetHeader(h).Code(&code).Do()

	assert.NoError(t, err)
}

type testQuery2 struct {
	Q8 uint8 `query:"q8" form:"q8"`
}

type testQuery struct {
	Q1 string    `query:"q1" form:"q1"`
	Q2 int       `query:"q2" form:"q2"`
	Q3 float32   `query:"q3" form:"q3"`
	Q4 float64   `query:"q4" form:"q4"`
	Q5 time.Time `query:"q5" form:"q5" time_format:"unix" time_location:"Asia/Shanghai"`
	Q6 time.Time `query:"q6" form:"q6" time_format:"unixNano" time_location:"Asia/Shanghai"`
	Q7 time.Time `query:"q7" form:"q7" time_format:"2006-01-02" time_location:"Asia/Shanghai"`
	testQuery2
}

func queryDefault() *testQuery {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		panic(err.Error())
	}

	return &testQuery{
		Q1: "v1",
		Q2: 2,
		Q3: 3.14,
		Q4: 3.1415,
		Q5: time.Date(2019, 7, 28, 14, 36, 0, 0, loc),
		Q6: time.Date(2019, 7, 28, 14, 36, 0, 1000, loc),
		Q7: time.Date(2019, 7, 28, 0, 0, 0, 0, loc),
		testQuery2: testQuery2{
			Q8: 8,
		},
	}
}

func TestSetQueryStruct(t *testing.T) {
	q := queryDefault()
	router := func() *gin.Engine {
		router := gin.Default()
		router.GET("/test.query", func(c *gin.Context) {
			q2 := testQuery{}
			err := c.BindQuery(&q2)
			assert.NoError(t, err)

			testQueryEqual(t, *q, q2)
			//assert.Equal(t, q, &q2)
		})

		return router
	}()

	ts := httptest.NewServer(http.HandlerFunc(router.ServeHTTP))
	defer ts.Close()

	g := New(nil)
	code := 0

	err := g.GET(ts.URL + "/test.query").SetQuery(q).Code(&code).Do()

	assert.NoError(t, err)
}

func TestQueryString(t *testing.T) {
	s := "q1=v1&q2=2&q3=3.14&q4=3.1415&q5=1564295760&q6=1564295760000001000&q7=2019-07-28&q8=8"
	testQueryStringCore(t, s, false)
	testQueryStringCore(t, s, true)
}

func testQueryEqual(t *testing.T, q1, q2 testQuery) {
	//不用assert.Equal(t, q1, q2)
	//assert.Equal 有个bug

	assert.Equal(t, q1.Q1, q2.Q1)
	assert.Equal(t, q1.Q2, q2.Q2)
	assert.Equal(t, q1.Q3, q2.Q3)
	assert.Equal(t, q1.Q4, q2.Q4)
	assert.Equal(t, q1.Q8, q2.Q8)
	if !q1.Q5.Equal(q2.Q5) {
		t.Errorf("want(%s) got(%s)\n", q1.Q5, q2.Q5)
	}
	if !q1.Q6.Equal(q2.Q6) {
		t.Errorf("want(%s) got(%s)\n", q1.Q6, q2.Q6)
	}
	if !q1.Q7.Equal(q2.Q7) {
		t.Errorf("want(%s) got(%s)\n", q1.Q7, q2.Q7)
	}
}

func testQueryStringCore(t *testing.T, qStr string, isPtr bool) {
	q := queryDefault()
	router := func() *gin.Engine {
		router := gin.Default()
		router.GET("/test.query", func(c *gin.Context) {
			q2 := testQuery{}
			err := c.BindQuery(&q2)

			//todo
			//fmt.Printf("------->q7(%t)\n", reflect.DeepEqual(q1.Q5, q2.Q5), reflect.DeepEqual(q1.Q6, q2.Q6), reflect.DeepEqual(q1.Q7, q2.Q7))
			//fmt.Printf("%s:%s, %t\n", q1.Q7, q2.Q7, q1.Q7.Equal(q2.Q7))
			assert.NoError(t, err)

			testQueryEqual(t, *q, q2)
		})

		return router
	}()

	ts := httptest.NewServer(http.HandlerFunc(router.ServeHTTP))
	defer ts.Close()

	g := New(nil)
	code := 0

	var err error
	if isPtr {
		err = g.GET(ts.URL + "/test.query").SetQuery(&qStr).Code(&code).Do()
	} else {
		err = g.GET(ts.URL + "/test.query").SetQuery(qStr).Code(&code).Do()
	}

	assert.NoError(t, err)
	assert.Equal(t, code, 200)
}

func setupQuery(t *testing.T, q *testQuery) func() *gin.Engine {
	return func() *gin.Engine {
		router := gin.Default()
		router.GET("/test.query", func(c *gin.Context) {
			q2 := testQuery{}
			err := c.BindQuery(&q2)
			assert.NoError(t, err)

			testQueryEqual(t, *q, q2)
			//assert.Equal(t, *q, q2)
		})

		return router
	}
}

func testQuerySliceAndArrayCore(t *testing.T, x interface{}) {
	q := queryDefault()
	router := setupQuery(t, q)

	ts := httptest.NewServer(http.HandlerFunc(router().ServeHTTP))
	defer ts.Close()

	g := New(nil)

	code := 0
	err := g.GET(ts.URL + "/test.query").SetQuery(x).Code(&code).Do()

	assert.NoError(t, err)
	assert.Equal(t, code, 200)
}

func testQueryFail(t *testing.T, x interface{}) {
	q := queryDefault()
	router := setupQuery(t, q)

	ts := httptest.NewServer(http.HandlerFunc(router().ServeHTTP))
	defer ts.Close()

	g := New(nil)

	code := 0
	err := g.GET(ts.URL + "/test.query").SetQuery(x).Code(&code).Do()
	assert.Error(t, err)
	assert.NotEqual(t, code, 200)
}

func TestQueryFail(t *testing.T) {
	testQueryFail(t, []string{"q1"})
}

func TestQuerySliceAndArray(t *testing.T) {
	q := queryDefault()
	testQuerySliceAndArrayCore(t, []string{"q1", "v1", "q2", "2", "q3", "3.14", "q4", "3.1415", "q5",
		fmt.Sprint(q.Q5.Unix()), "q6", fmt.Sprint(q.Q6.UnixNano()), "q7", q.Q7.Format("2006-01-02"), "q8", "8"})
	testQuerySliceAndArrayCore(t, [8 * 2]string{"q1", "v1", "q2", "2", "q3", "3.14", "q4", "3.1415", "q5",
		fmt.Sprint(q.Q5.Unix()), "q6", fmt.Sprint(q.Q6.UnixNano()), "q7", q.Q7.Format("2006-01-02"), "q8", "8"})

	testQuerySliceAndArrayCore(t, &[]string{"q1", "v1", "q2", "2", "q3", "3.14", "q4", "3.1415", "q5",
		fmt.Sprint(q.Q5.Unix()), "q6", fmt.Sprint(q.Q6.UnixNano()), "q7", q.Q7.Format("2006-01-02"), "q8", "8"})
	testQuerySliceAndArrayCore(t, &[8 * 2]string{"q1", "v1", "q2", "2", "q3", "3.14", "q4", "3.1415", "q5",
		fmt.Sprint(q.Q5.Unix()), "q6", fmt.Sprint(q.Q6.UnixNano()), "q7", q.Q7.Format("2006-01-02"), "q8", "8"})
}

type testBodyNeed struct {
	Float32 bool `form:"float32"`
	Float64 bool `form:"float64"`
	Uint    bool `form:"uint"`
	Uint8   bool `form:"uint8"`
	Uint16  bool `form:"uint16"`
	Uint32  bool `form:"uint32"`
	Uint64  bool `form:"uint64"`
	Int     bool `form:"int"`
	Int8    bool `form:"int8"`
	Int16   bool `form:"int16"`
	Int32   bool `form:"int32"`
	Int64   bool `form:"int64"`
	String  bool `form:"string"`
	Bytes   bool `form:"bytes"`
	Reader  bool `form:"reader"`
}

type testBodyBind struct {
	Type string `uri:"type"`
}

type testBodyReq struct {
	url  string
	got  interface{}
	need interface{}
}

func TestBindBody(t *testing.T) {
	router := func() *gin.Engine {
		router := gin.Default()

		bodyBind := testBodyBind{}

		router.GET("/:type", func(c *gin.Context) {
			c.ShouldBindUri(&bodyBind)

			switch bodyBind.Type {
			case "uint":
				c.String(200, "1")
			case "uint8":
				c.String(200, "2")
			case "uint16":
				c.String(200, "3")
			case "uint32":
				c.String(200, "4")
			case "uint64":
				c.String(200, "5")
			case "int":
				c.String(200, "6")
			case "int8":
				c.String(200, "7")
			case "int16":
				c.String(200, "8")
			case "int32":
				c.String(200, "9")
			case "int64":
				c.String(200, "10")
			case "float32":
				c.String(200, "11")
			case "float64":
				c.String(200, "12")
			case "string":
				c.String(200, "string")
			case "bytes":
				c.String(200, "bytes")
			case "io.writer":
				c.String(200, "io.writer")
			default:
				c.String(500, "unknown")
			}
		})

		return router
	}()

	ts := httptest.NewServer(http.HandlerFunc(router.ServeHTTP))

	tests := []testBodyReq{
		{url: "/uint", got: new(uint), need: core.NewPtrVal(uint(1))},
		{url: "/uint8", got: new(uint8), need: core.NewPtrVal(uint8(2))},
		{url: "/uint16", got: new(uint16), need: core.NewPtrVal(uint16(3))},
		{url: "/uint32", got: new(uint32), need: core.NewPtrVal(uint32(4))},
		{url: "/uint64", got: new(uint64), need: core.NewPtrVal(uint64(5))},
		{url: "/int", got: new(int), need: core.NewPtrVal(int(6))},
		{url: "/int8", got: new(int8), need: core.NewPtrVal(int8(7))},
		{url: "/int16", got: new(int16), need: core.NewPtrVal(int16(8))},
		{url: "/int32", got: new(int32), need: core.NewPtrVal(int32(9))},
		{url: "/int64", got: new(int64), need: core.NewPtrVal(int64(10))},
		{url: "/float32", got: new(float32), need: core.NewPtrVal(float32(11))},
		{url: "/float64", got: new(float64), need: core.NewPtrVal(float64(12))},
		{url: "/string", got: new(string), need: core.NewPtrVal("string")},
		{url: "/bytes", got: new([]byte), need: core.NewPtrVal([]byte("bytes"))},
		{url: "/io.writer", got: bytes.NewBufferString(""), need: bytes.NewBufferString("io.writer")},
	}

	for _, v := range tests {

		code := 0
		err := New(nil).GET(ts.URL + v.url).BindBody(v.got).Code(&code).Do()
		assert.Equal(t, code, 200)
		assert.NoError(t, err)
		assert.Equal(t, v.got, v.need)
	}

}

func TestSetBody(t *testing.T) {

	router := func() *gin.Engine {
		router := gin.Default()
		router.POST("/", func(c *gin.Context) {

			testBody := testBodyNeed{}

			c.ShouldBindQuery(&testBody)

			var s string
			b := bytes.NewBuffer(nil)
			io.Copy(b, c.Request.Body)
			defer c.Request.Body.Close()

			s = b.String()
			switch {
			case testBody.Int:
				assert.Equal(t, s, "1")
			case testBody.Int8:
				assert.Equal(t, s, "2")
			case testBody.Int16:
				assert.Equal(t, s, "3")
			case testBody.Int32:
				assert.Equal(t, s, "4")
			case testBody.Int64:
				assert.Equal(t, s, "5")
			case testBody.Uint:
				assert.Equal(t, s, "6")
			case testBody.Uint8:
				assert.Equal(t, s, "7")
			case testBody.Uint16:
				assert.Equal(t, s, "8")
			case testBody.Uint32:
				assert.Equal(t, s, "9")
			case testBody.Uint64:
				assert.Equal(t, s, "10")
			case testBody.String:
				assert.Equal(t, s, "test string")
			case testBody.Bytes:
				assert.Equal(t, s, "test bytes")
			case testBody.Float32:
				assert.Equal(t, s, "11")
			case testBody.Float64:
				assert.Equal(t, s, "12")
			case testBody.Reader:
				assert.Equal(t, s, "test io.Reader")
			default:
				c.JSON(500, "unknown type")
			}

		})

		return router
	}()

	ts := httptest.NewServer(http.HandlerFunc(router.ServeHTTP))

	code := 0
	err := New(nil).POST(ts.URL).SetQuery(H{"int": true}).SetBody(1).Code(&code).Do()
	assert.NoError(t, err)
	assert.Equal(t, code, 200)

	err = New(nil).POST(ts.URL).SetQuery(H{"int8": true}).SetBody(int8(2)).Code(&code).Do()
	assert.NoError(t, err)
	assert.Equal(t, code, 200)

	err = New(nil).POST(ts.URL).SetQuery(H{"int16": true}).SetBody(int16(3)).Code(&code).Do()
	assert.NoError(t, err)
	assert.Equal(t, code, 200)

	err = New(nil).POST(ts.URL).SetQuery(H{"int32": true}).SetBody(int32(4)).Code(&code).Do()
	assert.NoError(t, err)
	assert.Equal(t, code, 200)

	err = New(nil).POST(ts.URL).SetQuery(H{"int64": true}).SetBody(int64(5)).Code(&code).Do()
	assert.NoError(t, err)
	assert.Equal(t, code, 200)
	//=====================uint start

	err = New(nil).POST(ts.URL).SetQuery(H{"uint": true}).SetBody(6).Code(&code).Do()
	assert.NoError(t, err)
	assert.Equal(t, code, 200)

	err = New(nil).POST(ts.URL).SetQuery(H{"uint8": true}).SetBody(uint8(7)).Code(&code).Do()
	assert.NoError(t, err)
	assert.Equal(t, code, 200)

	err = New(nil).POST(ts.URL).SetQuery(H{"uint16": true}).SetBody(uint16(8)).Code(&code).Do()
	assert.NoError(t, err)
	assert.Equal(t, code, 200)

	err = New(nil).POST(ts.URL).SetQuery(H{"uint32": true}).SetBody(uint32(9)).Code(&code).Do()
	assert.NoError(t, err)
	assert.Equal(t, code, 200)

	err = New(nil).POST(ts.URL).SetQuery(H{"uint64": true}).SetBody(uint64(10)).Code(&code).Do()
	assert.NoError(t, err)
	assert.Equal(t, code, 200)
	//============================== float start

	err = New(nil).POST(ts.URL).SetQuery(H{"float32": true}).SetBody(float32(11)).Code(&code).Do()
	assert.NoError(t, err)
	assert.Equal(t, code, 200)

	err = New(nil).POST(ts.URL).SetQuery(H{"float64": true}).SetBody(float64(12)).Code(&code).Do()

	err = New(nil).POST(ts.URL).SetQuery(H{"string": true}).SetBody("test string").Code(&code).Do()
	assert.NoError(t, err)
	assert.Equal(t, code, 200)

	// test bytes string
	err = New(nil).POST(ts.URL).SetQuery(H{"bytes": true}).SetBody([]byte("test bytes")).Code(&code).Do()
	assert.NoError(t, err)
	assert.Equal(t, code, 200)

	// test io.Reader
	err = New(nil).POST(ts.URL).SetQuery(H{"reader": true}).SetBody(bytes.NewBufferString("test io.Reader")).Code(&code).Do()
	assert.NoError(t, err)
	assert.Equal(t, code, 200)
}

func setupProxy(t *testing.T) *gin.Engine {
	r := gin.Default()

	r.GET("/:a", func(c *gin.Context) {
		all, err := ioutil.ReadAll(c.Request.Body)

		assert.NoError(t, err)
		c.String(200, string(all))
	})

	return r
}

func TestProxy(t *testing.T) {
	router := setupProxy(t)
	ts := httptest.NewServer(http.HandlerFunc(router.ServeHTTP))
	defer ts.Close()
	proxyTs := httptest.NewServer(http.HandlerFunc(router.ServeHTTP))
	defer proxyTs.Close()

	code := 0
	var s string

	c := http.Client{}

	err := New(&c).GET(ts.URL + "/login").SetBody(proxyTs.URL).SetProxy(proxyTs.URL).BindBody(&s).Code(&code).Do()

	assert.NoError(t, err)
	assert.Equal(t, 200, code)
	assert.Equal(t, s, proxyTs.URL)

	// test fail
	err = New(&c).GET(ts.URL + "/login").SetProxy("\x7f" /*url.Parse源代码写了遇到\x7f会报错*/).Do()
	assert.Error(t, err)

	// 错误情况1
	c.Transport = &TransportFail{}
	err = New(&c).GET(ts.URL + "/login").SetProxy(proxyTs.URL).Do()
	assert.Error(t, err)

}

func setupCookie(t *testing.T, total *int32) *gin.Engine {

	router := gin.Default()

	router.GET("/cookie", func(c *gin.Context) {

		cookie1, err := c.Request.Cookie("test1")

		assert.NoError(t, err)
		assert.Equal(t, cookie1.Name, "test1")
		assert.Equal(t, cookie1.Value, "test1")

		cookie2, err := c.Request.Cookie("test2")
		assert.NoError(t, err)
		assert.Equal(t, cookie2.Name, "test2")
		assert.Equal(t, cookie2.Value, "test2")

		atomic.AddInt32(total, 1)

	})

	router.GET("/cookie/one", func(c *gin.Context) {

		cookie3, err := c.Request.Cookie("test3")

		assert.NoError(t, err)
		assert.Equal(t, cookie3.Name, "test3")
		assert.Equal(t, cookie3.Value, "test3")
		atomic.AddInt32(total, 1)

	})

	return router
}

func TestCookie(t *testing.T) {
	var total int32
	router := setupCookie(t, &total)

	ts := httptest.NewServer(http.HandlerFunc(router.ServeHTTP))

	err := GET(ts.URL+"/cookie").SetCookies(&http.Cookie{Name: "test1", Value: "test1"},
		&http.Cookie{Name: "test2", Value: "test2"}).Do()

	assert.NoError(t, err)
	err = GET(ts.URL + "/cookie/one").SetCookies(&http.Cookie{Name: "test3", Value: "test3"}).Do()

	assert.Equal(t, total, int32(2))
}

// Server side testing context function
func setupContext(t *testing.T) *gin.Engine {
	router := gin.New()

	router.GET("/cancel", func(c *gin.Context) {
		ctx := c.Request.Context()
		select {
		case <-ctx.Done():
			fmt.Printf("cancel done\n")
		case <-time.After(2 * time.Second):
			assert.Fail(t, "test cancel fail")
		}
	})

	router.GET("/timeout", func(c *gin.Context) {
		ctx := c.Request.Context()
		select {
		case <-ctx.Done():
			fmt.Printf("ctx timeout done\n")
		case <-time.After(2 * time.Second):
			assert.Fail(t, "test ctx timeout fail")
		}
	})

	return router
}

// test timeout
func testWithContextTimeout(t *testing.T, ts *httptest.Server) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*1)

	err := GET(ts.URL + "/timeout").WithContext(ctx).Do()
	assert.Error(t, err)
}

// test cancel
func testWithContextCancel(t *testing.T, ts *httptest.Server) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(time.Second)
		cancel()
	}()

	err := GET(ts.URL + "/cancel").WithContext(ctx).Do()
	assert.Error(t, err)
}

//
func TestWithContext(t *testing.T) {
	router := setupContext(t)
	ts := httptest.NewServer(http.HandlerFunc(router.ServeHTTP))

	testWithContextTimeout(t, ts)
	testWithContextCancel(t, ts)
}

func setupUnixSocket(t *testing.T, path string) *http.Server {
	router := gin.Default()
	type testHeader struct {
		H1 string `header:"h1"`
		H2 string `header:"h2"`
	}

	router.POST("/test/unix", func(c *gin.Context) {

		tHeader := testHeader{}
		err := c.ShouldBindHeader(&tHeader)

		assert.Equal(t, tHeader.H1, "v1")
		assert.Equal(t, tHeader.H2, "v2")
		assert.NoError(t, err)

		c.String(200, "ok")
	})

	listener, err := net.Listen("unix", path)
	assert.NoError(t, err)

	srv := http.Server{Handler: router}
	go func() {
		srv.Serve(listener)
	}()

	return &srv
}

type TransportFail struct{}

func (t *TransportFail) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, errors.New("fail")
}

func TestUnixSocket(t *testing.T) {
	path := "./unix.sock"
	defer os.Remove(path)

	ctx, cancel := context.WithCancel(context.Background())
	srv := setupUnixSocket(t, path)
	defer func() {
		srv.Shutdown(ctx)
		cancel()
	}()

	c := http.Client{}
	s := ""
	err := New(&c).UnixSocket(path).POST("http://xxx/test/unix/").SetHeader(H{"h1": "v1", "h2": "v2"}).BindBody(&s).Do()
	assert.NoError(t, err)
	assert.Equal(t, s, "ok")

	// 错误情况1
	c.Transport = &TransportFail{}
	err = New(&c).UnixSocket(path).POST("http://xxx/test/unix/").SetHeader(H{"h0": "v1", "h2": "v2"}).BindBody(&s).Do()
	assert.Error(t, err)
}

func setupDebug(t *testing.T) *gin.Engine {
	r := gin.New()

	r.GET("/", func(c *gin.Context) {
		all, err := ioutil.ReadAll(c.Request.Body)

		assert.NoError(t, err)
		c.String(200, string(all))
	})

	return r
}

func TestDebug(t *testing.T) {
	buf := &bytes.Buffer{}

	router := setupDebug(t)
	ts := httptest.NewServer(http.HandlerFunc(router.ServeHTTP))

	color.NoColor = false
	test := []func() DebugOpt{
		// 测试颜色
		func() DebugOpt {
			return DebugFunc(func(o *DebugOption) {
				buf.Reset()
				o.Debug = true
				o.Color = true
				o.Write = buf
			})
		},

		// 测试打开日志输出
		func() DebugOpt {
			return DebugFunc(func(o *DebugOption) {
				t.Logf("--->1.DebugOption address = %p\n", o)
				o.Debug = true
			})
		},

		// 测试修改输出源
		func() DebugOpt {
			return DebugFunc(func(o *DebugOption) {
				t.Logf("--->2.DebugOption address = %p\n", o)
				buf.Reset()
				o.Debug = true
				o.Write = buf
			})
		},

		// 测试环境变量
		func() DebugOpt {
			return DebugFunc(func(o *DebugOption) {
				buf.Reset()
				if len(os.Getenv("IOS_DEBUG")) > 0 {
					o.Debug = true
				}
				o.Write = buf
			})
		},

		// 没有颜色输出
		NoColor,
	}

	s := ""
	os.Setenv("IOS_DEBUG", "true")
	for k, v := range test {
		s = ""
		err := GET(ts.URL).
			Debug(v()).
			SetBody(fmt.Sprintf("%d test debug.", k)).
			BindBody(&s).
			Do()
		assert.NoError(t, err)

		if k != 0 {
			assert.NotEqual(t, buf.Len(), 0)
		}

		assert.Equal(t, fmt.Sprintf("%d test debug.", k), s)
	}

	err := GET(ts.URL).Debug(true).SetBody("true test debug").BindBody(&s).Do()

	assert.NoError(t, err)
	assert.Equal(t, s, "true test debug")
}

type testWWWForm struct {
	Int     int     `form:"int" www-form:"int"`
	Float64 float64 `form:"float64" www-form:"float64"`
	String  string  `form:"string" www-form:"string"`
}

func setupWWWForm(t *testing.T, need testWWWForm) *gin.Engine {
	r := gin.New()

	r.POST("/", func(c *gin.Context) {
		wf := testWWWForm{}

		err := c.ShouldBind(&wf)

		assert.NoError(t, err)
		//err := c.ShouldBind(&wf)
		assert.Equal(t, need, wf)
	})

	return r
}

func TestWWWForm(t *testing.T) {
	need := testWWWForm{
		Int:     3,
		Float64: 3.14,
		String:  "test-www-Form",
	}

	router := setupWWWForm(t, need)
	ts := httptest.NewServer(http.HandlerFunc(router.ServeHTTP))

	err := POST(ts.URL).Debug(true).SetWWWForm(need).Do()
	assert.NoError(t, err)
}

func setup_DataFlow(t *testing.T) *gin.Engine {
	router := gin.New()

	router.GET("/timeout", func(c *gin.Context) {
		ctx := c.Request.Context()
		select {
		case <-ctx.Done():
			fmt.Printf("setTimeout done\n")
		case <-time.After(2 * time.Second):
			assert.Fail(t, "test timeout fail")
		}
	})

	return router
}

func Test_DataFlow_Timeout(t *testing.T) {
	router := setup_DataFlow(t)
	ts := httptest.NewServer(http.HandlerFunc(router.ServeHTTP))

	const (
		longTimeout   = 400
		middleTimeout = 300
		shortTimeout  = 200
	)
	// 只设置timeout
	err := GET(ts.URL + "/timeout").
		SetTimeout(shortTimeout * time.Millisecond).
		Do()
	assert.Error(t, err)

	// 使用互斥api的原则，后面的覆盖前面的
	// 这里是SetTimeout生效, 超时时间200ms
	ctx, _ := context.WithTimeout(context.Background(), longTimeout*time.Millisecond)
	s := time.Now()
	err = GET(ts.URL + "/timeout").
		WithContext(ctx).
		SetTimeout(shortTimeout * time.Millisecond).
		Do()

	assert.Error(t, err)

	assert.LessOrEqual(t, int(time.Now().Sub(s)), int(middleTimeout*time.Millisecond))

	// 使用互斥api的原则，后面的覆盖前面的
	// 这里是WithContext生效, 超时时间400ms
	ctx, _ = context.WithTimeout(context.Background(), longTimeout*time.Millisecond)
	s = time.Now()
	err = GET(ts.URL + "/timeout").
		SetTimeout(shortTimeout * time.Millisecond).
		WithContext(ctx).
		Do()

	assert.Error(t, err)
	assert.GreaterOrEqual(t, int(time.Now().Sub(s)), int(middleTimeout*time.Millisecond))
}
