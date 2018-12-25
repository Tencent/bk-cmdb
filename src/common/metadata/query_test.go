package metadata

import (
	"testing"
)

func TestSearchSortParseStr(t *testing.T) {
	type testData struct {
		input string
		ss    []SearchSort
	}
	testDataArr := []testData{
		testData{
			input: "aa,bb",
			ss: []SearchSort{
				SearchSort{
					Field: "aa",
				},
				SearchSort{
					Field: "bb",
				},
			},
		},
		testData{
			input: "aa:-1,bb:1,cc,dd:abc",
			ss: []SearchSort{
				SearchSort{
					Field: "aa",
					IsDsc: true,
				},
				SearchSort{
					Field: "bb",
				},
				SearchSort{
					Field: "cc",
				},
				SearchSort{
					Field: "dd",
				},
			},
		},
	}
	for _, testDataItem := range testDataArr {
		testSSArr := NewSearchSortParse().String(testDataItem.input).ToSearchSortArr()
		if len(testSSArr) != len(testDataItem.ss) {
			t.Errorf("str parse to Search Sort error!")
			return
		}
		for idx, ssItem := range testSSArr {
			if ssItem.Field != testDataItem.ss[idx].Field ||
				ssItem.IsDsc != testDataItem.ss[idx].IsDsc {
				t.Errorf("%+v, %+v not equal", ssItem, testDataItem.ss[idx])
				return
			}
		}
	}

}

func TestSearchSortToMongo(t *testing.T) {
	type testData struct {
		input  []SearchSort
		output string
	}
	testDataArr := []testData{
		testData{
			output: "aa:1,bb:1",
			input: []SearchSort{
				SearchSort{
					Field: "aa",
				},
				SearchSort{
					Field: "bb",
				},
			},
		},
		testData{
			output: "aa:-1,bb:1,cc:1,dd:1",
			input: []SearchSort{
				SearchSort{
					Field: "aa",
					IsDsc: true,
				},
				SearchSort{
					Field: "bb",
				},
				SearchSort{
					Field: "cc",
				},
				SearchSort{
					Field: "dd",
				},
			},
		},
	}
	for _, testDataItem := range testDataArr {
		orderBy := NewSearchSortParse().Set(testDataItem.input).ToMongo()
		if orderBy != testDataItem.output {
			t.Errorf("%s != %s", orderBy, testDataItem.output)
			return
		}

	}

}

func TestSearchSortField(t *testing.T) {
	type testData struct {
		ss []SearchSort
	}
	testDataArr := []testData{
		testData{
			ss: []SearchSort{
				SearchSort{
					Field: "aa",
				},
				SearchSort{
					Field: "bb",
				},
			},
		},
		testData{
			ss: []SearchSort{
				SearchSort{
					Field: "aa",
					IsDsc: true,
				},
				SearchSort{
					Field: "bb",
				},
				SearchSort{
					Field: "cc",
				},
				SearchSort{
					Field: "dd",
				},
			},
		},
	}
	for _, testDataItem := range testDataArr {
		ssHandle := NewSearchSortParse()
		for _, ssItem := range testDataItem.ss {
			ssHandle.Field(ssItem.Field, ssItem.IsDsc)
		}
		testSSArr := ssHandle.ToSearchSortArr()
		if len(testSSArr) != len(testDataItem.ss) {
			t.Errorf("str parse to Search Sort error!")
			return
		}
		for idx, ssItem := range testSSArr {
			if ssItem.Field != testDataItem.ss[idx].Field ||
				ssItem.IsDsc != testDataItem.ss[idx].IsDsc {
				t.Errorf("%+v, %+v not equal", ssItem, testDataItem.ss[idx])
				return
			}
		}
	}

}
