package access_decision

/*

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/vera/x/acp/auth_engine"
	"github.com/sourcenetwork/vera/x/acp/policy"
	"github.com/sourcenetwork/vera/x/acp/testutil"
	"github.com/sourcenetwork/vera/x/acp/types"
)

var timestamp = testutil.MustDateTimeToProto("2023-07-26 14:08:30")

func setupEvaluateAccessRequest(t *testing.T) (context.Context, auth_engine.AuthEngine, Repository, ParamsRepository, *types.Policy) {
	polStr := `
    id: unregister-pol
    resources:
      file:
        permissions:
          read:
            expr: owner + reader + admin
        relations:
          owner:
            types:
              - actor
          reader:
            types:
              - actor
              - file->reader
          admin:
            manages:
              - reader
            types:
              - actor

    actor:
      name: actor
    `
	polIR, err := policy.Unmarshal(polStr, types.PolicyMarshalingType_YAML)
	require.Nil(t, err)

	engine, _, ctx := testutil.GetTestAuthEngine(t)

	createCmd := policy.CreatePolicyCommand{
		Policy:       polIR,
		Creator:      "cosmos1gue5de6a8fdff0jut08vw5sg9pk6rr00cstakj",
		CreationTime: timestamp,
	}
	db := dbm.NewMemDB()
	pol, err := createCmd.Execute(ctx, db, engine)
	require.Nil(t, err)

	_, err = engine.SetRelationship(ctx, pol, &types.RelationshipRecord{
		Relationship: types.NewActorRelationship("file", "readme.txt", "owner", "alice"),
		Archived:     false,
		Creator:      "alice",
	})
	require.Nil(t, err)

	_, err = engine.SetRelationship(ctx, pol, &types.RelationshipRecord{
		Relationship: types.NewActorRelationship("file", "alice.txt", "owner", "alice"),
		Archived:     false,
		Creator:      "alice",
	})
	require.Nil(t, err)

	_, err = engine.SetRelationship(ctx, pol, &types.RelationshipRecord{
		Relationship: types.NewActorRelationship("file", "readme.txt", "reader", "bob"),
		Archived:     false,
		Creator:      "alice",
	})
	require.Nil(t, err)

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db = dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())
	kv := stateStore.GetCommitKVStore(storeKey)
	require.NotNil(t, kv)
	repository := NewAccessDecisionRepository(kv)

	paramRepo := &StaticParamsRepository{}
	return ctx, engine, repository, paramRepo, pol
}


func TestEvaluateAccessRequest_GeneratesAccessDecisionWhenActorIsAuthorized(t *testing.T) {
	ctx, engine, repo, paramRepo, pol := setupEvaluateAccessRequest(t)

	cmd := EvaluateAccessRequestsCommand{
		Policy: pol,
        Operations: []*types.Operation{
            &types.Operation{
                Object:     types.NewObject("file", "readme.txt"),
                Permission: "read",
            },
        },
        Actor: &types.Actor{
            Id: "bob",
        },
		CreationTime:  timestamp,
		Creator:       "creator",
		CurrentHeight: 1,
	}
	decision, err := cmd.Execute(ctx, engine, repo, paramRepo)

	want := &types.AccessDecision{
		Id:            "837adcadec6abc793fea5819330d54c568a1d2f7989c7363b2ecb3fd88636756",
		AccessRequest: cmd.AccessRequest,
		Params: &types.DecisionParams{
			ExpirationHeightDelta: 100,
		},
		CreationTime: timestamp,
		IssuedHeight: 1,
		Creator:      "creator",
	}
	require.Nil(t, err)
	require.Equal(t, decision, want)
}

func TestEvaluateAccessRequest_ReturnsUnauthorizedErrorWhenActorIsNotAllowedSomeOperation(t *testing.T) {
	ctx, engine, repo, paramRepo, pol := setupEvaluateAccessRequest(t)

	cmd := EvaluateAccessRequestsCommand{
		Policy: pol,
		AccessRequest: &types.AccessRequest{
			Operations: []*types.Operation{
				&types.Operation{
					Object:     types.NewObject("file", "readme.txt"),
					Permission: "read",
				},
				&types.Operation{
					Object:     types.NewObject("file", "alice.txt"),
					Permission: "read",
				},
			},
			Actor: &types.Actor{
				Id: "bob",
			},
		},
		CreationTime:  timestamp,
		Creator:       "creator",
		CurrentHeight: 1,
	}
	decision, err := cmd.Execute(ctx, engine, repo, paramRepo)

	require.Nil(t, decision)
	require.ErrorIs(t, err, types.ErrNotAuthorized)
}

*/
