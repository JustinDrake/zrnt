package debug_json

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/protolambda/go-beacon-transition/eth2/util/ssz"
	"reflect"
	"regexp"
	"strings"
)

func encode(v reflect.Value) interface{} {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Struct:
		return encodeToMap(v)
	case reflect.Slice, reflect.Array:
		if v.Type().Elem().Kind() == reflect.Struct {
			return encodeToStructList(v)
		}
		// return an explicit empty list for length 0 elements (may be nil slice)
		if v.Len() == 0 {
			return make([]interface{}, 0)
		}
		return v.Interface()
	default:
		return v.Interface()
	}
}

type OrderedMapEntry struct {
	Key   string
	Value interface{}
}
type OrderedMap []OrderedMapEntry

func (ordMap OrderedMap) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")
	length := len(ordMap)
	count := 0
	for _, kv := range ordMap {
		jsonValue, err := json.Marshal(kv.Value)
		if err != nil {
			return nil, err
		}
		buffer.WriteString(fmt.Sprintf("\"%s\":%s", kv.Key, string(jsonValue)))
		count++
		if count < length {
			buffer.WriteString(",")
		}
	}
	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func encodeToStructList(v reflect.Value) []interface{} {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	items := v.Len()
	out := make([]interface{}, 0, items)
	for i := 0; i < items; i++ {
		out = append(out, encode(v.Index(i)))
	}
	return out
}

func encodeToMap(v reflect.Value) OrderedMap {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	fields := v.NumField()
	out := make(OrderedMap, 0, fields)
	for i := 0; i < fields; i++ {
		f := v.Field(i)
		// ignore unexported struct fields
		if !f.CanSet() {
			continue
		}
		// Transform the name to the python formatting: snake case
		name := toSnakeCase(v.Type().Field(i).Name)
		out = append(out, OrderedMapEntry{name, encode(f)})
		fieldHashTreeRoot := ssz.Hash_tree_root(f.Interface())
		encodedHashTreeRoot := make([]byte, hex.EncodedLen(len(fieldHashTreeRoot)))
		hex.Encode(encodedHashTreeRoot, fieldHashTreeRoot[:])
		out = append(out, OrderedMapEntry{name + "_hash_tree_root", string(encodedHashTreeRoot)})
	}
	return out
}

func EncodeToTreeRootJSON(v interface{}, indent string) ([]byte, error) {
	preEncoded := encode(reflect.ValueOf(v))

	// add "hash_tree_root" to top level map
	if asOrdMap, ok := preEncoded.(OrderedMap); ok {
		coreHashTreeRoot := ssz.Hash_tree_root(v)
		encodedHashTreeRoot := make([]byte, hex.EncodedLen(len(coreHashTreeRoot)))
		hex.Encode(encodedHashTreeRoot, coreHashTreeRoot[:])
		preEncoded = append(asOrdMap, OrderedMapEntry{"hash_tree_root", string(encodedHashTreeRoot)})
	}
	if indent != "" {
		return json.MarshalIndent(preEncoded, "", indent)
	} else {
		return json.Marshal(preEncoded)
	}
}
