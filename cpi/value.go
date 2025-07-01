package cpi

import (
	"bytes"
	"fmt"
	"slices"
	"strconv"
)

const (
	KTypeNumber = iota
	KTypeString
	KTypeNil
	KTypeBool
	KTypeDict
	KTypeList
	KTypeFunction
)

var TypeNames [7]string

func init() {
	TypeNames[0] = "number"
	TypeNames[1] = "string"
	TypeNames[2] = "nil"
	TypeNames[3] = "bool"
	TypeNames[4] = "dict"
	TypeNames[5] = "list"
	TypeNames[6] = "function"
}

type KValue interface {
	Type() int
	Str() string
}

type KString string

func kString(v string) KString {
	return KString(v)
}

func (k KString) Type() int {
	return KTypeString
}

func (k KString) Str() string {
	return string(k)
}

type KNumber float64

func kNumber(v string) KNumber {
	value, err := strconv.ParseFloat(v, 64)
	if err != nil {
		panic(err)
	}
	return KNumber(value)
}

func (k KNumber) Type() int {
	return KTypeNumber
}

func (k KNumber) Str() string {
	return fmt.Sprintf("%.2f", k)
}

func compareKValue(a, b KValue) bool {
	if a.Type() != b.Type() {
		return false
	}
	return a == b
}

type KNil struct{}

func (k KNil) Type() int {
	return KTypeNil
}

func (k KNil) Str() string {
	return "nil"
}

type strlist struct {
	array []string
}
type KDict struct {
	dict map[string]KValue
	keys *strlist
}

func NewKDict(cap int) KDict {
	return KDict{
		dict: make(map[string]KValue),
		keys: &strlist{
			array: make([]string, 0, cap),
		},
	}
}

func (d KDict) Type() int {
	return KTypeDict
}

func (d KDict) Str() string {
	buffer := bytes.NewBuffer([]byte("{"))
	l := len(d.dict)
	idx := 0
	for k, v := range d.dict {
		buffer.WriteString(fmt.Sprintf(" %s:%s", k, v.Str()))
		if idx < l-1 {
			buffer.WriteRune(',')
		}
		idx++
	}
	buffer.WriteRune('}')
	return buffer.String()
}

func (d KDict) GetField(field string) KValue {
	v, ok := d.dict[field]
	if ok {
		return v
	}
	return KNil{}
}

func (d KDict) SetField(field string, value KValue) {
	d.dict[field] = value
	if !slices.Contains(d.keys.array, field) {
		d.keys.array = append(d.keys.array, field)
	}
}

func (d KDict) GetAt(index int) KValue {
	if len(d.keys.array) <= index {
		panic("index out of len dict")
	}
	k := d.keys.array[index]
	return d.dict[k]
}

func (d KDict) GetKeyValue(index int) (string, KValue) {
	if len(d.keys.array) <= index {
		panic("index out of len dict")
	}
	key := d.keys.array[index]
	return key, d.dict[key]
}

func (d KDict) Len() int { return len(d.keys.array) }

type KList struct {
	list *klist
}

func (l KList) Type() int {
	return KTypeList
}

func (k KList) Str() string {
	buffer := bytes.NewBuffer([]byte("["))
	l := len(k.list.array)
	for idx, e := range k.list.array {
		buffer.WriteString(e.Str())
		if idx < l-1 {
			buffer.WriteRune(',')
		}
	}
	buffer.WriteRune(']')
	return buffer.String()
}

type klist struct {
	array []KValue
}

func NewKList(cap int) KList {
	return KList{
		list: &klist{
			array: make([]KValue, 0, cap),
		},
	}
}

func (l KList) AppendArray(arr []KValue) {
	l.list.array = append(l.list.array, arr...)
}

func (l KList) Append(v KValue) {
	l.list.array = append(l.list.array, v)
}

func (l KList) GetAt(index int) KValue {
	return l.list.array[index]
}

func (l KList) SetAt(index int, v KValue) {
	if index == len(l.list.array) {
		l.list.array = append(l.list.array, v)
		return
	}
	// auto panic
	l.list.array[index] = v
}

func (l KList) Len() int {
	return len(l.list.array)
}

type KBool bool

func (b KBool) Type() int {
	return KTypeBool
}

func (b KBool) Str() string {
	return fmt.Sprintf("%t", b)
}
