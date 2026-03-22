package stores

import (
	"testing"

	"github.com/sourcenetwork/sourcehub/x/acp/types"
	"github.com/stretchr/testify/require"
)

func TestGogoProtoMarshalerRoundTrip(t *testing.T) {
	marshaler := NewGogoProtoMarshaler(func() *types.Params {
		return &types.Params{}
	})

	original := types.DefaultParams()
	origPtr := &original
	bytes, err := marshaler.Marshal(&origPtr)
	require.NoError(t, err)
	require.NotEmpty(t, bytes)

	decoded, err := marshaler.Unmarshal(bytes)
	require.NoError(t, err)
	require.Equal(t, original.PolicyCommandMaxExpirationDelta, decoded.PolicyCommandMaxExpirationDelta)
}

func TestGogoProtoMarshalerUnmarshalInvalid(t *testing.T) {
	marshaler := NewGogoProtoMarshaler(func() *types.Params {
		return &types.Params{}
	})

	_, err := marshaler.Unmarshal([]byte("invalid-protobuf-data"))
	require.Error(t, err)
}
