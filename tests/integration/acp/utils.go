package test

import (
	"errors"
	"reflect"
	"time"

	gogotypes "github.com/cosmos/gogoproto/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MustDateTimeToProto parses a time.DateTime (YYYY-MM-DD HH:MM:SS) timestamp
// and converts into a proto Timestamp.
// Panics if input is invalid
func MustDateTimeToProto(timestamp string) *gogotypes.Timestamp {
	t, err := time.Parse(time.DateTime, timestamp)
	if err != nil {
		panic(err)
	}

	ts, err := gogotypes.TimestampProto(t)
	if err != nil {
		panic(err)
	}

	return ts
}

func TimeToProto(ts time.Time) *gogotypes.Timestamp {
	return &gogotypes.Timestamp{
		Seconds: ts.Unix(),
		Nanos:   0,
	}
}

func AssertResults(ctx *TestCtx, got, want any, gotErr, wantErr error) {
	if wantErr != nil {
		require.NotNil(ctx.T, gotErr, "expected an error but got none")
		if errors.Is(gotErr, wantErr) {
			assert.ErrorIs(ctx.T, gotErr, wantErr)
		} else {
			// Errors returned from SDK operations (RPC communication to a SourceHub node)
			// no longer have the original errors wrapped, therefore we compare a string as fallback strat.

			gotErrStr := gotErr.Error()
			wantErrStr := wantErr.Error()
			assert.Contains(ctx.T, gotErrStr, wantErrStr)
		}
	} else {
		assert.NoError(ctx.T, gotErr)
	}
	if !isNil(want) {
		normalizeTimestamps(got)
		normalizeTimestamps(want)
		assert.Equal(ctx.T, want, got)
	}
}

// Helper function to normalize timestamps by setting Nanos to 0 in any struct containing proto timestamps.
func normalizeTimestamps(obj any) {
	if obj == nil {
		return
	}

	// Check if obj is a pointer and dereference it for reflection.
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Only handle structs, slices, or maps for recursive normalization.
	switch val.Kind() {
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			if field.Kind() == reflect.Ptr && field.Type().String() == "*types.Timestamp" {
				if ts, ok := field.Interface().(*gogotypes.Timestamp); ok && ts != nil {
					ts.Nanos = 0
				}
			} else {
				normalizeTimestamps(field.Interface())
			}
		}
	case reflect.Slice:
		for i := 0; i < val.Len(); i++ {
			normalizeTimestamps(val.Index(i).Interface())
		}
	case reflect.Map:
		for _, key := range val.MapKeys() {
			normalizeTimestamps(val.MapIndex(key).Interface())
		}
	}
}

func isNil(object any) bool {
	if object == nil {
		return true
	}

	value := reflect.ValueOf(object)
	kind := value.Kind()
	isNilableKind := containsKind(
		[]reflect.Kind{
			reflect.Chan, reflect.Func,
			reflect.Interface, reflect.Map,
			reflect.Ptr, reflect.Slice, reflect.UnsafePointer},
		kind)

	if isNilableKind && value.IsNil() {
		return true
	}

	return false
}

// containsKind checks if a specified kind in the slice of kinds.
func containsKind(kinds []reflect.Kind, kind reflect.Kind) bool {
	for i := 0; i < len(kinds); i++ {
		if kind == kinds[i] {
			return true
		}
	}

	return false
}
