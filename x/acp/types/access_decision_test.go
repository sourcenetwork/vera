package types

import (
	"testing"
	"time"

	prototypes "github.com/cosmos/gogoproto/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/stretchr/testify/require"
)

func makeTestTimestamp(t *testing.T) *Timestamp {
	ts, err := prototypes.TimestampProto(time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	return NewTimestamp(ts, 100)
}

func TestAccessDecisionProduceId(t *testing.T) {
	creationTs := makeTestTimestamp(t)
	decision := &AccessDecision{
		PolicyId:           "policy-1",
		Creator:            "cosmos1creator",
		Actor:              "did:example:actor",
		CreatorAccSequence: 1,
		IssuedHeight:       100,
		CreationTime:       creationTs,
		Operations: []*coretypes.Operation{
			{
				Object:     coretypes.NewObject("resource", "obj1"),
				Permission: "read",
			},
		},
		Params: &DecisionParams{
			DecisionExpirationDelta: 10,
			ProofExpirationDelta:    5,
			TicketExpirationDelta:   20,
		},
	}

	id := decision.ProduceId()
	require.NotEmpty(t, id)

	// deterministic
	id2 := decision.ProduceId()
	require.Equal(t, id, id2)
}

func TestAccessDecisionProduceIdDifferentInputs(t *testing.T) {
	creationTs := makeTestTimestamp(t)
	params := &DecisionParams{
		DecisionExpirationDelta: 10,
		ProofExpirationDelta:    5,
		TicketExpirationDelta:   20,
	}

	d1 := &AccessDecision{
		PolicyId:     "policy-1",
		Creator:      "cosmos1a",
		Actor:        "did:example:a",
		CreationTime: creationTs,
		Operations: []*coretypes.Operation{
			{Object: coretypes.NewObject("res", "1"), Permission: "read"},
		},
		Params: params,
	}
	d2 := &AccessDecision{
		PolicyId:     "policy-2",
		Creator:      "cosmos1a",
		Actor:        "did:example:a",
		CreationTime: creationTs,
		Operations: []*coretypes.Operation{
			{Object: coretypes.NewObject("res", "1"), Permission: "read"},
		},
		Params: params,
	}

	require.NotEqual(t, d1.ProduceId(), d2.ProduceId())
}

func TestAccessDecisionHashParams(t *testing.T) {
	decision := &AccessDecision{
		Params: &DecisionParams{
			DecisionExpirationDelta: 10,
			ProofExpirationDelta:    5,
			TicketExpirationDelta:   20,
		},
	}
	hash := decision.hashParams()
	require.NotEmpty(t, hash)
	require.Len(t, hash, 32) // sha256
}
