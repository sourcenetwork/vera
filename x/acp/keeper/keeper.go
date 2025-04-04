package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	acpruntime "github.com/sourcenetwork/acp_core/pkg/runtime"
	"github.com/sourcenetwork/acp_core/pkg/services"
	"github.com/sourcenetwork/raccoondb/v2/primitives"
	"github.com/sourcenetwork/sourcehub/x/acp/capability"
	cosmosadapter "github.com/sourcenetwork/sourcehub/x/acp/stores/cosmos"

	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	"github.com/sourcenetwork/sourcehub/x/acp/access_decision"
	"github.com/sourcenetwork/sourcehub/x/acp/commitment"
	"github.com/sourcenetwork/sourcehub/x/acp/keeper/policy_cmd"
	"github.com/sourcenetwork/sourcehub/x/acp/registration"
	"github.com/sourcenetwork/sourcehub/x/acp/stores"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService store.KVStoreService
		logger       log.Logger

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string

		accountKeeper types.AccountKeeper
		capKeeper     *capabilitykeeper.ScopedKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	accountKeeper types.AccountKeeper,
	capKeeper *capabilitykeeper.ScopedKeeper,

) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return Keeper{
		cdc:           cdc,
		storeService:  storeService,
		authority:     authority,
		logger:        logger,
		accountKeeper: accountKeeper,
		capKeeper:     capKeeper,
	}
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// getAccessDecisionRepository returns the module's default access decision repository
func (k *Keeper) getAccessDecisionRepository(ctx sdk.Context) access_decision.Repository {
	kv := k.storeService.OpenKVStore(ctx)
	prefixKey := []byte(types.AccessDecisionRepositoryKeyPrefix)
	adapted := runtime.KVStoreAdapter(kv)
	adapted = prefix.NewStore(adapted, prefixKey)
	return access_decision.NewAccessDecisionRepository(adapted)
}

// getACPEngine returns the module's default ACP Core Engine
func (k *Keeper) getACPEngine(ctx sdk.Context) *services.EngineService {
	kv := k.storeService.OpenKVStore(ctx)
	adapted := runtime.KVStoreAdapter(kv)
	raccoonAdapted := stores.RaccoonKVFromCosmos(adapted)
	runtime, err := acpruntime.NewRuntimeManager(
		acpruntime.WithKVStore(raccoonAdapted),
		acpruntime.WithTimeService(&SourceHubTimeProvider{}),
	)
	if err != nil {
		panic(err)
	}
	return services.NewACPEngine(runtime)
}

// getRegistrationsCommitmentRepository returns the module's default Registration Commitment Repository
func (k *Keeper) getRegistrationsCommitmentRepository(ctx sdk.Context) *commitment.CommitmentRepository {
	cmtkv := k.storeService.OpenKVStore(ctx)
	kv := cosmosadapter.NewFromCoreKVStore(cmtkv)
	kv = primitives.NewPrefixedKV(kv, []byte(types.RegistrationsCommitmentKeyPrefix))
	repo, err := commitment.NewCommitmentRepository(kv)
	if err != nil {
		panic(err)
	}
	return repo
}

// getAmendmentEventRepository returns the module's default AmendmentEventRepository
func (k *Keeper) getAmendmentEventRepository(ctx sdk.Context) *registration.AmendmentEventRepository {
	cmtkv := k.storeService.OpenKVStore(ctx)
	kv := cosmosadapter.NewFromCoreKVStore(cmtkv)
	kv = primitives.NewPrefixedKV(kv, []byte(types.AmendmentEventKeyPrefix))
	repo, err := registration.NewAmendmentEventRepository(kv)
	if err != nil {
		panic(err)
	}
	return repo
}

// GetComitmentService returns the module's default CommitmentService instance
func (k *Keeper) getCommitmentService(ctx sdk.Context) *commitment.CommitmentService {
	return commitment.NewCommitmentService(
		k.getACPEngine(ctx),
		k.getRegistrationsCommitmentRepository(ctx),
	)
}

// getRegistrationService returns the module's default RegistrationService instance
func (k *Keeper) getRegistrationService(ctx sdk.Context) *registration.RegistrationService {
	return registration.NewRegistrationService(
		k.getACPEngine(ctx),
		k.getAmendmentEventRepository(ctx),
		k.getCommitmentService(ctx),
	)
}

// getPolicyCmdHandler returns the module's default PolicyCmd Handler instance
func (k *Keeper) getPolicyCmdHandler(ctx sdk.Context) *policy_cmd.Handler {
	return policy_cmd.NewPolicyCmdHandler(
		k.getACPEngine(ctx),
		k.getRegistrationService(ctx),
		k.getCommitmentService(ctx),
	)
}

func (k *Keeper) getPolicyCapabilityManager(ctx sdk.Context) *capability.PolicyCapabilityManager {
	return capability.NewPolicyCapabilityManager(k.capKeeper)
}

// InitializeCapabilityKeeper allows main app to set the capability
// keeper after the moment of creation.
//
// This is supported since currently the capability module
// does not integrate with the new module dependency injection system.
//
// If the keeper was previously initialized (ie inner point != nil),
// throws a panic
func (k *Keeper) InitializeCapabilityKeeper(keeper *capabilitykeeper.ScopedKeeper) {
	if k.capKeeper != nil {
		panic("capability keeper already initialized")
	}
	k.capKeeper = keeper
}
