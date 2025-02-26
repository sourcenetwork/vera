package test

import (
	"fmt"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sourcenetwork/sourcehub/x/acp/signed_policy_cmd"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

const (
	SourceHubAuthStratEnvVar string = "SOURCEHUB_ACP_TEST_AUTH"
	SourceHubExecutorEnvVar  string = "SOURCEHUB_ACP_TEST_EXECUTOR"
	SourceHubActorEnvVar     string = "SOURCEHUB_ACP_TEST_ACTOR"
)

type AccountCreator interface {
	// GetOrCreateActor retrieves an account from a TestActor's address
	// if the account does not exist in the chain, it must be created
	// and given credits (if required)
	GetOrCreateAccountFromActor(*TestCtx, *TestActor) (sdk.AccountI, error)
}

// ACPClient represents a component which can execute an ACP Msg and produce a result
type ACPClient interface {
	AccountCreator
	signed_policy_cmd.LogicalClock

	CreatePolicy(ctx *TestCtx, msg *types.MsgCreatePolicy) (*types.MsgCreatePolicyResponse, error)
	BearerPolicyCmd(ctx *TestCtx, msg *types.MsgBearerPolicyCmd) (*types.MsgBearerPolicyCmdResponse, error)
	SignedPolicyCmd(ctx *TestCtx, msg *types.MsgSignedPolicyCmd) (*types.MsgSignedPolicyCmdResponse, error)
	DirectPolicyCmd(ctx *TestCtx, msg *types.MsgDirectPolicyCmd) (*types.MsgDirectPolicyCmdResponse, error)

	Policy(ctx *TestCtx, msg *types.QueryPolicyRequest) (*types.QueryPolicyResponse, error)
	RegistrationsCommitmentByCommitment(ctx *TestCtx, msg *types.QueryRegistrationsCommitmentByCommitmentRequest) (*types.QueryRegistrationsCommitmentByCommitmentResponse, error)
	RegistrationsCommitment(ctx *TestCtx, msg *types.QueryRegistrationsCommitmentRequest) (*types.QueryRegistrationsCommitmentResponse, error)

	ObjectOwner(ctx *TestCtx, msg *types.QueryObjectOwnerRequest) (*types.QueryObjectOwnerResponse, error)

	// GetLastBlockTs returns an ACP Timestamp for the last accepted block in SourceHub
	GetLastBlockTs(ctx *TestCtx) (*types.Timestamp, error)

	Cleanup()
	// WaitBlock waits until the Executor has advanced to the next block
	WaitBlock()
}

// AuthenticationStrategy is an enum representing the Authentication format
// which should be used in the tests
type AuthenticationStrategy int

const (
	// Direct represents authentication done directly thought a Tx/Msg Signer
	Direct AuthenticationStrategy = iota
	// BearerToken auth uses a Bearer Token to authenticate the actor
	BearerToken
	// SignedPayload auth uses a SignedPolicyCmd as source of authentication
	SignedPayload
)

var AuthenticationStrategyMap map[string]AuthenticationStrategy = map[string]AuthenticationStrategy{
	"DIRECT": Direct,
	"BEARER": BearerToken,
	"SIGNED": SignedPayload,
}

// ActorKeyType represents the key pair to be used by the system Actors during the test
type ActorKeyType int

const (
	Actor_ED25519 ActorKeyType = iota
	Actor_SECP256K1
)

var ActorKeyMap map[string]ActorKeyType = map[string]ActorKeyType{
	"ED25519":   Actor_ED25519,
	"SECP256K1": Actor_SECP256K1,
}

// ExecutorStrategy represents the available executors for the test suite
type ExecutorStrategy int

const (
	// Keeper calls the ACP Keeper directly without going through a consensus engine
	Keeper ExecutorStrategy = iota
	// CLI invokes SourceHub through the CLI client and broadcasts the Msgs
	// to a running instance of SourceHub
	CLI
	// SDK broadcasts Msgs to a running instance of SourceHub through the SourceHub SDK Client
	SDK
)

var ExecutorStrategyMap map[string]ExecutorStrategy = map[string]ExecutorStrategy{
	"KEEPER": Keeper,
	//"CLI":    CLI,
	//"SDK": SDK,
}

// TestConfig models how the tests suite will be run
type TestConfig struct {
	AuthStrategy     AuthenticationStrategy
	ExecutorStrategy ExecutorStrategy
	ActorType        ActorKeyType
	//TODO InitialState
}

func NewDefaultTestConfig() TestConfig {
	return TestConfig{
		AuthStrategy:     BearerToken,
		ExecutorStrategy: Keeper,
		ActorType:        Actor_ED25519,
	}
}

func MustNewTestConfigFromEnv() TestConfig {
	config := NewDefaultTestConfig()

	actor, wasSet := os.LookupEnv(SourceHubActorEnvVar)
	if wasSet {
		key, found := ActorKeyMap[actor]
		if !found {
			panic(fmt.Errorf("ActorKey string value not defined: %v", actor))
		}
		config.ActorType = key
	}

	authStratStr, wasSet := os.LookupEnv(SourceHubAuthStratEnvVar)
	if wasSet {
		authStrat, found := AuthenticationStrategyMap[authStratStr]
		if !found {
			panic(fmt.Errorf("AuthenticationStrategy string value not defined: %v", authStratStr))
		}
		config.AuthStrategy = authStrat
	}

	executorStr, wasSet := os.LookupEnv(SourceHubExecutorEnvVar)
	if wasSet {
		executor, found := ExecutorStrategyMap[executorStr]
		if !found {
			panic(fmt.Errorf("ExecutorStrategy string value not defined: %v", executorStr))
		}
		config.ExecutorStrategy = executor
	}

	return config
}
