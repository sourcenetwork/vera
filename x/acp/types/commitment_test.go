package types

import (
	"testing"
	"time"

	prototypes "github.com/cosmos/gogoproto/types"
	"github.com/stretchr/testify/require"
)

func TestRegistrationsCommitmentIsExpiredAgainst(t *testing.T) {
	creationTime := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	creationProto, err := prototypes.TimestampProto(creationTime)
	require.NoError(t, err)

	commitment := &RegistrationsCommitment{
		Metadata: &RecordMetadata{
			CreationTs: NewTimestamp(creationProto, 100),
		},
		Validity: NewDurationFromTimeDuration(10 * time.Minute),
	}

	tests := []struct {
		name    string
		nowTime time.Time
		block   uint64
		expired bool
	}{
		{"not expired", creationTime.Add(5 * time.Minute), 105, false},
		{"at boundary", creationTime.Add(10 * time.Minute), 110, false},
		{"expired", creationTime.Add(11 * time.Minute), 111, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nowProto, err := prototypes.TimestampProto(tc.nowTime)
			require.NoError(t, err)
			now := NewTimestamp(nowProto, tc.block)
			result, err := commitment.IsExpiredAgainst(now)
			require.NoError(t, err)
			require.Equal(t, tc.expired, result)
		})
	}
}

func TestRegistrationsCommitmentIsExpiredAgainstBlockCount(t *testing.T) {
	commitment := &RegistrationsCommitment{
		Metadata: &RecordMetadata{
			CreationTs: NewTimestamp(nil, 100),
		},
		Validity: NewBlockCountDuration(50),
	}

	now := NewTimestamp(nil, 160)
	expired, err := commitment.IsExpiredAgainst(now)
	require.NoError(t, err)
	require.True(t, expired)

	now2 := NewTimestamp(nil, 140)
	expired2, err := commitment.IsExpiredAgainst(now2)
	require.NoError(t, err)
	require.False(t, expired2)
}
