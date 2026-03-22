package types

import (
	"testing"
	"time"

	prototypes "github.com/cosmos/gogoproto/types"
	"github.com/stretchr/testify/require"
)

func TestNewBlockCountDuration(t *testing.T) {
	d := NewBlockCountDuration(100)
	require.NotNil(t, d)
	bc, ok := d.Duration.(*Duration_BlockCount)
	require.True(t, ok)
	require.Equal(t, uint64(100), bc.BlockCount)
}

func TestNewDurationFromTimeDuration(t *testing.T) {
	d := NewDurationFromTimeDuration(5 * time.Minute)
	require.NotNil(t, d)
	pd, ok := d.Duration.(*Duration_ProtoDuration)
	require.True(t, ok)
	require.NotNil(t, pd.ProtoDuration)
}

func TestTimestampToISOString(t *testing.T) {
	goTime := time.Date(2024, time.March, 15, 10, 30, 0, 0, time.UTC)
	protoTs, err := prototypes.TimestampProto(goTime)
	require.NoError(t, err)

	ts := NewTimestamp(protoTs, 42)
	iso, err := ts.ToISOString()
	require.NoError(t, err)
	require.Equal(t, "2024-03-15T10:30:00Z", iso)
}

func TestTimestampIsAfterBlockCount(t *testing.T) {
	ts := NewTimestamp(nil, 100)
	duration := NewBlockCountDuration(50)

	tests := []struct {
		name     string
		nowBlock uint64
		expected bool
	}{
		{"before expiry", 140, false},
		{"at expiry", 150, false},
		{"after expiry", 151, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			now := NewTimestamp(nil, tc.nowBlock)
			result, err := ts.IsAfter(duration, now)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestTimestampIsAfterProtoDuration(t *testing.T) {
	goTime := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	protoTs, err := prototypes.TimestampProto(goTime)
	require.NoError(t, err)

	ts := NewTimestamp(protoTs, 0)
	duration := NewDurationFromTimeDuration(1 * time.Hour)

	tests := []struct {
		name     string
		nowTime  time.Time
		expected bool
	}{
		{"before expiry", time.Date(2024, time.January, 1, 0, 30, 0, 0, time.UTC), false},
		{"at expiry", time.Date(2024, time.January, 1, 1, 0, 0, 0, time.UTC), false},
		{"after expiry", time.Date(2024, time.January, 1, 1, 0, 1, 0, time.UTC), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nowProto, err := prototypes.TimestampProto(tc.nowTime)
			require.NoError(t, err)
			now := NewTimestamp(nowProto, 0)
			result, err := ts.IsAfter(duration, now)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestTimestampIsAfterNilDurationPanics(t *testing.T) {
	ts := NewTimestamp(nil, 100)
	now := NewTimestamp(nil, 200)
	duration := &Duration{Duration: nil}

	require.Panics(t, func() {
		ts.IsAfter(duration, now)
	})
}

func TestNewTimestamp(t *testing.T) {
	goTime := time.Date(2024, time.June, 1, 0, 0, 0, 0, time.UTC)
	protoTs, err := prototypes.TimestampProto(goTime)
	require.NoError(t, err)

	ts := NewTimestamp(protoTs, 42)
	require.Equal(t, uint64(42), ts.BlockHeight)
	require.Equal(t, protoTs, ts.ProtoTs)
}
