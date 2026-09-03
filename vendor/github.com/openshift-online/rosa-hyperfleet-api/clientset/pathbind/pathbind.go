// Package pathbind maps between flat consumer structs and deeply-nested SDK types
// using dotted JSON-path tags on the consumer struct fields.
//
// Consumer structs declare mappings via the hfsdk: struct tag:
//
//	type MyInput struct {
//	    Region string  `hfsdk:"spec.hostedCluster.platform.aws.region"`
//	    Name   string  `hfsdk:"metadata.name"`
//	    Derived string `hfsdk:"-"` // skipped; set manually after Expand
//	}
//
// Expand populates an SDK struct from the tagged fields of a consumer struct.
// Flatten reads tagged consumer struct fields back from an SDK struct.
// Both support native Go scalar types, pointer-to-scalar, and metav1.Time.
package pathbind

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Expand populates dst (a non-nil pointer to an SDK struct) from the hfsdk:-tagged
// fields of src (a struct or pointer to struct).
// Intermediate pointer-to-struct fields in dst are allocated if nil.
// Fields tagged hfsdk:"-" and zero/nil fields in src are skipped.
func Expand(_ context.Context, src any, dst any) error {
	srcVal := indirect(reflect.ValueOf(src))
	dstVal := indirect(reflect.ValueOf(dst))
	if !srcVal.IsValid() || srcVal.Kind() != reflect.Struct {
		return fmt.Errorf("pathbind.Expand: src must be a struct or pointer to struct")
	}
	if !dstVal.IsValid() || dstVal.Kind() != reflect.Struct || !dstVal.CanSet() {
		return fmt.Errorf("pathbind.Expand: dst must be a non-nil pointer to a struct")
	}

	srcType := srcVal.Type()
	for i := 0; i < srcType.NumField(); i++ {
		field := srcType.Field(i)
		tag := field.Tag.Get("hfsdk")
		if tag == "" || tag == "-" {
			continue
		}

		goValue, isSet := unwrap(srcVal.Field(i))
		if !isSet {
			continue
		}

		if err := setAtPath(dstVal, strings.Split(tag, "."), goValue); err != nil {
			return fmt.Errorf("pathbind.Expand: field %s (path %q): %w", field.Name, tag, err)
		}
	}
	return nil
}

// Flatten reads hfsdk:-tagged fields of dst (a non-nil pointer to a consumer struct)
// from src (a struct or pointer to struct).
// Intermediate nil pointers in src are treated as absent (skipped, not an error).
// Fields tagged hfsdk:"-" are skipped.
func Flatten(_ context.Context, src any, dst any) error {
	srcVal := indirect(reflect.ValueOf(src))
	dstVal := indirect(reflect.ValueOf(dst))
	if !srcVal.IsValid() || srcVal.Kind() != reflect.Struct {
		return fmt.Errorf("pathbind.Flatten: src must be a struct or pointer to struct")
	}
	if !dstVal.IsValid() || dstVal.Kind() != reflect.Struct || !dstVal.CanSet() {
		return fmt.Errorf("pathbind.Flatten: dst must be a non-nil pointer to a struct")
	}

	dstType := dstVal.Type()
	for i := 0; i < dstType.NumField(); i++ {
		field := dstType.Field(i)
		tag := field.Tag.Get("hfsdk")
		if tag == "" || tag == "-" {
			continue
		}

		goValue, err := getAtPath(srcVal, strings.Split(tag, "."))
		if err != nil {
			return fmt.Errorf("pathbind.Flatten: field %s (path %q): %w", field.Name, tag, err)
		}
		if goValue == nil {
			continue
		}

		converted, err := toFieldValue(goValue, field.Type)
		if err != nil {
			return fmt.Errorf("pathbind.Flatten: field %s: %w", field.Name, err)
		}
		dstVal.Field(i).Set(converted)
	}
	return nil
}

// unwrap extracts a native Go value from a consumer struct field.
// Returns (value, true) when the field holds a meaningful value; (nil, false) when it is
// absent (nil pointer, empty string, or zero for non-pointer scalars).
// Boolean false is considered meaningful and returns (false, true).
// Explicit zero values in non-nil pointers are preserved and returned as present.
func unwrap(v reflect.Value) (any, bool) {
	if !v.IsValid() {
		return nil, false
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, false
		}
		// For non-nil pointers to numeric types, preserve explicit zero values.
		elem := v.Elem()
		switch elem.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return elem.Int(), true
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return elem.Uint(), true
		case reflect.Float32, reflect.Float64:
			return elem.Float(), true
		}
		// For other pointer types, recurse.
		return unwrap(elem)
	}
	switch v.Kind() {
	case reflect.String:
		s := v.String()
		if s == "" {
			return nil, false
		}
		return s, true
	case reflect.Bool:
		return v.Bool(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n := v.Int()
		if n == 0 {
			return nil, false
		}
		return n, true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n := v.Uint()
		if n == 0 {
			return nil, false
		}
		return n, true
	case reflect.Float32, reflect.Float64:
		f := v.Float()
		if f == 0 {
			return nil, false
		}
		return f, true
	case reflect.Slice:
		if v.IsNil() || v.Len() == 0 {
			return nil, false
		}
		return v.Interface(), true
	case reflect.Map:
		if v.IsNil() || v.Len() == 0 {
			return nil, false
		}
		return v.Interface(), true
	}
	if v.Type() == reflect.TypeOf(metav1.Time{}) {
		t := v.Interface().(metav1.Time)
		if t.IsZero() {
			return nil, false
		}
		return t.UTC().Format(time.RFC3339), true
	}
	if v.Kind() == reflect.Struct {
		return v.Interface(), true
	}
	return nil, false
}

// setAtPath traverses segments in v, allocating nil pointer-to-struct intermediates,
// and sets the leaf to goValue converted to the leaf field's type.
func setAtPath(v reflect.Value, segments []string, goValue any) error {
	cur := v
	for i, seg := range segments {
		fv, ok := findField(cur, seg)
		if !ok {
			path := strings.Join(segments[:i], ".")
			if path != "" {
				path += "."
			}
			return fmt.Errorf("segment %q not found in %s at path %q", seg, cur.Type(), path+seg)
		}
		if i == len(segments)-1 {
			converted, err := toFieldValue(goValue, fv.Type())
			if err != nil {
				return fmt.Errorf("at segment %q: %w", seg, err)
			}
			fv.Set(converted)
			return nil
		}
		// Intermediate: dereference pointer, allocating if nil.
		if fv.Kind() == reflect.Pointer {
			if fv.IsNil() {
				fv.Set(reflect.New(fv.Type().Elem()))
			}
			cur = fv.Elem()
		} else {
			cur = fv
		}
	}
	return nil
}

// getAtPath traverses segments in v and returns the leaf value.
// Returns (nil, nil) when any intermediate pointer is nil (absent, not an error).
func getAtPath(v reflect.Value, segments []string) (any, error) {
	cur := v
	for i, seg := range segments {
		fv, ok := findField(cur, seg)
		if !ok {
			path := strings.Join(segments[:i], ".")
			if path != "" {
				path += "."
			}
			return nil, fmt.Errorf("segment %q not found in %s at path %q", seg, cur.Type(), path+seg)
		}
		if i == len(segments)-1 {
			val := fromSDKValue(fv)
			return val, nil
		}
		if fv.Kind() == reflect.Pointer {
			if fv.IsNil() {
				return nil, nil
			}
			cur = fv.Elem()
		} else {
			cur = fv
		}
	}
	return nil, nil
}

// findField looks up a struct field by its json: tag name.
// Falls back to searching inline-embedded (anonymous or ,inline-tagged) fields
// to handle types like metav1.TypeMeta `json:",inline"`.
// Returns no match (false) if v is not a struct (including after pointer dereference).
func findField(v reflect.Value, seg string) (reflect.Value, bool) {
	// Guard: v must be a struct to have fields.
	if v.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}

	t := v.Type()
	// Direct fields first.
	for i := 0; i < t.NumField(); i++ {
		name, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
		if name == seg {
			return v.Field(i), true
		}
	}
	// Inline / anonymous embedded fields.
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if !field.Anonymous && !strings.HasPrefix(jsonTag, ",inline") {
			continue
		}
		fv := v.Field(i)
		if fv.Kind() == reflect.Pointer {
			if fv.IsNil() {
				continue
			}
			fv = fv.Elem()
		}
		if fv.Kind() != reflect.Struct {
			continue
		}
		if found, ok := findField(fv, seg); ok {
			return found, true
		}
	}
	return reflect.Value{}, false
}

// toFieldValue converts a native Go value (from unwrap or fromSDKValue) to a
// reflect.Value of targetType. Handles pointer wrapping, int-width conversion,
// and string-to-named-string-type conversion (e.g. string → PlatformType).
func toFieldValue(goValue any, targetType reflect.Type) (reflect.Value, error) {
	src := reflect.ValueOf(goValue)
	if !src.IsValid() {
		return reflect.Zero(targetType), nil
	}

	// Pointer target: convert element, then wrap in a new pointer.
	if targetType.Kind() == reflect.Pointer {
		inner, err := toFieldValue(goValue, targetType.Elem())
		if err != nil {
			return reflect.Value{}, err
		}
		ptr := reflect.New(targetType.Elem())
		ptr.Elem().Set(inner)
		return ptr, nil
	}

	// metav1.Time target from RFC3339 string.
	if targetType == reflect.TypeOf(metav1.Time{}) {
		s, ok := goValue.(string)
		if !ok {
			return reflect.Value{}, fmt.Errorf("expected string for metav1.Time, got %T", goValue)
		}
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("cannot parse %q as RFC3339: %w", s, err)
		}
		return reflect.ValueOf(metav1.NewTime(t)), nil
	}

	if src.Type().AssignableTo(targetType) {
		return src, nil
	}
	// ConvertibleTo handles: int64→int32, string→PlatformType (named string type), etc.
	// Reject numeric-to-string conversions (would reinterpret as Unicode code points).
	if src.Type().ConvertibleTo(targetType) {
		// Reject numeric → string conversions that would reinterpret values.
		if targetType.Kind() == reflect.String && isNumericKind(src.Kind()) {
			return reflect.Value{}, fmt.Errorf("cannot convert numeric %v to string (would reinterpret as code point)", src.Type())
		}
		// Validate integer narrowing conversions (e.g. int64→int32) don't overflow.
		if isSignedIntKind(src.Kind()) && isSignedIntKind(targetType.Kind()) {
			if !isValueInRange(src.Int(), targetType) {
				return reflect.Value{}, fmt.Errorf("integer value %d out of range for %s", src.Int(), targetType)
			}
		} else if isUnsignedIntKind(src.Kind()) && isUnsignedIntKind(targetType.Kind()) {
			if !isValueInRangeUnsigned(src.Uint(), targetType) {
				return reflect.Value{}, fmt.Errorf("integer value %d out of range for %s", src.Uint(), targetType)
			}
		}
		return src.Convert(targetType), nil
	}

	// JSON round-trip fallback for mismatched complex types.
	// string → slice/map/struct:  JSON unmarshal the source string.
	// slice/map/struct → string:  JSON marshal the source value.
	if goStr, ok := goValue.(string); ok {
		switch targetType.Kind() {
		case reflect.Slice, reflect.Map, reflect.Struct:
			ptr := reflect.New(targetType)
			if err := json.Unmarshal([]byte(goStr), ptr.Interface()); err != nil {
				return reflect.Value{}, fmt.Errorf("JSON unmarshal into %s: %w", targetType, err)
			}
			return ptr.Elem(), nil
		}
	}
	if targetType.Kind() == reflect.String {
		switch src.Kind() {
		case reflect.Slice, reflect.Map, reflect.Struct:
			b, err := json.Marshal(goValue)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("JSON marshal %T: %w", goValue, err)
			}
			return reflect.ValueOf(string(b)), nil
		}
	}

	return reflect.Value{}, fmt.Errorf("cannot convert %T to %s", goValue, targetType)
}

// fromSDKValue extracts a plain Go value from an SDK field value.
// Dereferences pointers; nil pointers return nil.
// metav1.Time is returned as an RFC3339 string.
// Named string types (e.g. PlatformType) are returned as plain string.
// Slices, maps, and non-metav1.Time structs are returned as their native Go
// value; toFieldValue handles the final conversion (e.g. JSON-marshal to string
// when the consumer field is a string).
func fromSDKValue(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		return fromSDKValue(v.Elem())
	}
	if v.Type() == reflect.TypeOf(metav1.Time{}) {
		t := v.Interface().(metav1.Time)
		if t.IsZero() {
			return nil
		}
		return t.UTC().Format(time.RFC3339)
	}
	if v.Kind() == reflect.String {
		return v.String()
	}
	if v.Kind() == reflect.Slice && v.IsNil() {
		return nil
	}
	if v.Kind() == reflect.Map && v.IsNil() {
		return nil
	}
	return v.Interface()
}

// indirect dereferences a chain of pointers to reach the underlying value.
func indirect(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

// isNumericKind checks if a reflect.Kind is numeric (int, uint, or float).
func isNumericKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// isSignedIntKind checks if a reflect.Kind is a signed integer type.
func isSignedIntKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	default:
		return false
	}
}

// isUnsignedIntKind checks if a reflect.Kind is an unsigned integer type.
func isUnsignedIntKind(k reflect.Kind) bool {
	switch k {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

// isValueInRange checks if a signed integer value fits in the target type without overflow.
func isValueInRange(val int64, targetType reflect.Type) bool {
	switch targetType.Kind() {
	case reflect.Int8:
		return val >= math.MinInt8 && val <= math.MaxInt8
	case reflect.Int16:
		return val >= math.MinInt16 && val <= math.MaxInt16
	case reflect.Int32:
		return val >= math.MinInt32 && val <= math.MaxInt32
	case reflect.Int64, reflect.Int:
		return true // no overflow for int64 or int
	default:
		return false
	}
}

// isValueInRangeUnsigned checks if an unsigned integer value fits in the target type without overflow.
func isValueInRangeUnsigned(val uint64, targetType reflect.Type) bool {
	switch targetType.Kind() {
	case reflect.Uint8:
		return val <= math.MaxUint8
	case reflect.Uint16:
		return val <= math.MaxUint16
	case reflect.Uint32:
		return val <= math.MaxUint32
	case reflect.Uint64, reflect.Uint:
		return true // no overflow for uint64 or uint
	default:
		return false
	}
}
