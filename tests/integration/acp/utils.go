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

// AssertError asserts that got and want match
// if want is not nil.
// If ant is nil, it asserts that got has no error
func AssertError(ctx *TestCtx, got, want error) bool {
	if want != nil {
		require.NotNil(ctx.T, got, "expected an error but got none")
		if errors.Is(got, want) {
			return assert.ErrorIs(ctx.T, got, want)
		} else {
			// Errors returned from SDK operations (RPC communication to a SourceHub node)
			// no longer have the original errors wrapped, therefore we compare a string as fallback strat.
			gotErrStr := got.Error()
			wantErrStr := want.Error()
			return assert.Contains(ctx.T, gotErrStr, wantErrStr)
		}
	} else {
		return assert.NoError(ctx.T, got)
	}
}

// AssertValue asserts got matches want, if want is not nil
func AssertValue(ctx *TestCtx, got, want any) {
	if !isNil(want) {
		assert.Equal(ctx.T, want, got)
	}
}

func AssertResults(ctx *TestCtx, got, want any, gotErr, wantErr error) {
	AssertError(ctx, gotErr, wantErr)
	AssertValue(ctx, got, want)
}

func isNil(object interface{}) bool {
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
