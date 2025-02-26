package types

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	prototypes "github.com/cosmos/gogoproto/types"
)

// NewBlockCountDuration returns a Duration object
// based on number of blocks
func NewBlockCountDuration(blocks uint64) *Duration {
	return &Duration{
		Duration: &Duration_BlockCount{
			BlockCount: blocks,
		},
	}
}

// NewDurationFromTimeDuration returns a new Duration object
// based on wall time
func NewDurationFromTimeDuration(duration time.Duration) *Duration {
	return &Duration{
		Duration: &Duration_ProtoDuration{
			ProtoDuration: prototypes.DurationProto(duration),
		},
	}
}

// ToISOString returns an ISO8601 timestamp
func (ts *Timestamp) ToISOString() (string, error) {
	t, err := prototypes.TimestampFromProto(ts.ProtoTs)
	if err != nil {
		return "", err
	}
	return t.Format(time.RFC3339), nil
}

// IsAfter returns whether now is after ts + duration
func (ts *Timestamp) IsAfter(duration *Duration, now *Timestamp) (bool, error) {
	switch d := duration.Duration.(type) {
	case *Duration_BlockCount:
		return ts.BlockHeight+d.BlockCount < now.BlockHeight, nil
	case *Duration_ProtoDuration:
		goTs, err := prototypes.TimestampFromProto(ts.ProtoTs)
		if err != nil {
			return false, err
		}

		goNow, err := prototypes.TimestampFromProto(now.ProtoTs)
		if err != nil {
			return false, err
		}

		goDuration, err := prototypes.DurationFromProto(d.ProtoDuration)
		if err != nil {
			return false, err
		}

		return goNow.After(goTs.Add(goDuration)), nil
	default:
		panic("invalid duration")
	}
}

// TimestampFromCtx returns a new Timestamp
// from a Cosmos Ctx
func TimestampFromCtx(ctx sdk.Context) (*Timestamp, error) {
	ts, err := prototypes.TimestampProto(ctx.BlockTime())
	if err != nil {
		return nil, err
	}
	return NewTimestamp(ts, uint64(ctx.BlockHeight())), nil
}

// NewTimestamp returns a Timestamp from a protobuff timestamp and block height
func NewTimestamp(time *prototypes.Timestamp, height uint64) *Timestamp {
	return &Timestamp{
		BlockHeight: height,
		ProtoTs:     time,
	}
}
