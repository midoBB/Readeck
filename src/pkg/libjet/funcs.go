// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package libjet

import (
	"fmt"
	"hash/adler32"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/CloudyKit/jet/v6"
	"github.com/readeck/readeck/pkg/utils"
)

var strType = reflect.TypeOf("")

var funcMap = map[string]jet.Func{
	"string": func(a jet.Arguments) reflect.Value {
		a.RequireNumOfArguments("string", 1, 1)
		return reflect.ValueOf(ToString(a.Get(0)))
	},
	"empty": jet.Func(func(a jet.Arguments) reflect.Value {
		a.RequireNumOfArguments("empty", 1, 1)
		return reflect.ValueOf(IsEmpty(a.Get(0)))
	}),
	"default": func(a jet.Arguments) reflect.Value {
		a.RequireNumOfArguments("default", 2, 2)
		if ToString(a.Get(0)) == "" {
			return a.Get(1)
		}
		return a.Get(0)
	},
	"join": func(a jet.Arguments) reflect.Value {
		a.RequireNumOfArguments("join", 2, 2)
		vl, isNil := Indirect(a.Get(0))
		if isNil {
			return reflect.ValueOf("")
		}
		list, ok := vl.([]string)
		if !ok {
			panic("invalid list type in join()")
		}
		sep := ToString(a.Get(1))

		return reflect.ValueOf(strings.Join(list, sep))
	},
	"date": func(a jet.Arguments) reflect.Value {
		a.RequireNumOfArguments("date", 2, 2)
		return reflect.ValueOf(ToDateFmt(a.Get(0), a.Get(1)))
	},
	"checksum": func(a jet.Arguments) reflect.Value {
		a.RequireNumOfArguments("checksum", 1, 1)
		h := adler32.New()
		h.Write([]byte(ToString(a.Get(0))))

		return reflect.ValueOf(fmt.Sprintf("%x", h.Sum32()))
	},
	"shortText": func(args jet.Arguments) reflect.Value {
		args.RequireNumOfArguments("shortText", 2, 2)
		s := ToString(args.Get(0))
		maxChars := ToInt(args.Get(1))

		return reflect.ValueOf(utils.ShortText(s, maxChars))
	},
	"shortURL": func(args jet.Arguments) reflect.Value {
		args.RequireNumOfArguments("shortText", 2, 2)
		s := ToString(args.Get(0))
		maxChars := ToInt(args.Get(1))

		return reflect.ValueOf(utils.ShortURL(s, maxChars))
	},
	"attrList": func(args jet.Arguments) reflect.Value {
		if args.NumOfArguments()%2 > 0 {
			panic("attrList(): incomplete key-value pair")
		}

		res := attrList{}

		for i := 0; i < args.NumOfArguments(); i += 2 {
			k := args.Get(i)
			v := args.Get(i + 1)
			if !k.IsValid() {
				args.Panicf("attrList(): key argument at position %d is not a valid value!", i)
			}
			if !v.IsValid() {
				args.Panicf("attrList(): key argument at position %d is not a valid value!", i+1)
			}
			if !k.Type().ConvertibleTo(strType) {
				args.Panicf("attrList(): can't use %+v as string key: %s is not convertible to string", k, k.Type())
			}
			if !v.Type().ConvertibleTo(strType) {
				args.Panicf("attrList(): can't use %+v as string key: %s is not convertible to string", v, v.Type())
			}

			val, isNil := Indirect(v)
			if !isNil {
				res[k.String()] = []any{val}
			}
		}

		return reflect.ValueOf(res)
	},
}

// FuncMap returns the jet function map.
func FuncMap() map[string]jet.Func {
	return funcMap
}

func VarMap() map[string]interface{} {
	return map[string]interface{}{
		"unsafeWrite": func(src io.Reader) jet.RendererFunc {
			return func(r *jet.Runtime) {
				io.Copy(r.Writer, src)
			}
		},
	}
}

// AddFuncToSet adds a given function to a jet.Set template set.
func AddFuncToSet(set *jet.Set, key string) {
	if f, ok := funcMap[key]; ok {
		set.AddGlobalFunc(key, f)
	}
}

// Indirect returns the underlying value of a reflect.Value.
// It resolves pointers and indicates if the value is nil.
func Indirect(val reflect.Value) (interface{}, bool) {
	switch val.Kind() {
	case reflect.Invalid:
		return nil, true
	case reflect.Ptr, reflect.Interface:
		if val.IsNil() {
			return nil, true
		}
		return Indirect(val.Elem())
	case reflect.Slice, reflect.Map, reflect.Func, reflect.Chan:
		if val.IsNil() {
			return nil, true
		}
	}

	return val.Interface(), false
}

// IsEmpty returns true if the value is considered empty.
func IsEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String, reflect.Array, reflect.Map, reflect.Slice:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Invalid:
		return true
	case reflect.Interface, reflect.Ptr:
		if v.IsNil() {
			return true
		}
		return IsEmpty(v.Elem())
	case reflect.Struct:
		val, _ := Indirect(v)
		if t, ok := val.(time.Time); ok && t.IsZero() {
			return true
		}
	}
	return false
}

// ToString converts a value to a string.
func ToString(v reflect.Value) string {
	val, isNil := Indirect(v)
	if isNil || val == nil {
		return ""
	}

	switch v := val.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	}

	if val, ok := val.(fmt.Stringer); ok {
		return val.String()
	}

	return fmt.Sprintf("%v", val)
}

// ToDateFmt returns a date formatted with the given format.
func ToDateFmt(d reflect.Value, f reflect.Value) string {
	v, isNil := Indirect(d)
	if isNil {
		return ""
	}
	date, ok := v.(time.Time)
	if !ok {
		panic("first argument must be a time.Time value or pointer")
	}

	layout := ToString(f)
	return date.Format(layout)
}

func ToInt(v reflect.Value) int {
	val, isNil := Indirect(v)
	if isNil || val == nil {
		return 0
	}

	switch v := val.(type) {
	case float32, float64:
		return int(v.(float64))
	case int:
		return v
	}

	panic("value is not a number")
}

type attrList map[string][]any

func (l attrList) Render(r *jet.Runtime) {
	i := 0
	for k, values := range l {
		if len(values) == 1 {
			if x, ok := values[0].(bool); ok && x {
				r.Write([]byte(k))
				continue
			} else if ok && !x {
				continue
			}
		}

		r.Writer.Write([]byte(k + `="`))
		for j, x := range values {
			v, err := getString(x)
			if err != nil {
				panic(err)
			}
			r.Write([]byte(v))
			if j+1 < len(values) {
				r.Write([]byte(" "))
			}
		}
		r.Writer.Write([]byte(`"`))
		if i+1 < len(l) {
			r.Write([]byte(" "))
		}
		i++
	}
}

func (l attrList) Add(key string, value any) {
	l[key] = append(l[key], value)
}

func (l attrList) Set(key string, value any) {
	l[key] = []any{value}
}

func getString(input any) (string, error) {
	switch v := input.(type) {
	case string:
		return v, nil
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v), nil
	case float32, float64:
		return fmt.Sprintf("%f", v), nil
	case bool:
		if v {
			return "true", nil
		}
		return "false", nil
	default:
		if s, ok := input.(fmt.Stringer); ok {
			return s.String(), nil
		}
	}

	return "", fmt.Errorf(`cannot convert "%v"`, input)
}
