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
	cosmosadapter "github.com/sourcenetwork/sourcehub/x/acp/stores/cosmos"

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
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	accountKeeper types.AccountKeeper,

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

// GetAccessDecisionRepository returns the module's default access decision repository
func (k *Keeper) GetAccessDecisionRepository(ctx sdk.Context) access_decision.Repository {
	kv := k.storeService.OpenKVStore(ctx)
	prefixKey := []byte(types.AccessDecisionRepositoryKeyPrefix)
	adapted := runtime.KVStoreAdapter(kv)
	adapted = prefix.NewStore(adapted, prefixKey)
	return access_decision.NewAccessDecisionRepository(adapted)
}

// GetACPEngine returns the module's default ACP Core Engine
func (k *Keeper) GetACPEngine(ctx sdk.Context) *services.EngineService {
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

// GetRegistrationsCommitmentRepository returns the module's default Registration Commitment Repository
func (k *Keeper) GetRegistrationsCommitmentRepository(ctx sdk.Context) *commitment.CommitmentRepository {
	cmtkv := k.storeService.OpenKVStore(ctx)
	kv := cosmosadapter.NewFromCoreKVStore(cmtkv)
	kv = primitives.NewPrefixedKV(kv, []byte(types.RegistrationsCommitmentKeyPrefix))
	repo, err := commitment.NewCommitmentRepository(kv)
	if err != nil {
		panic(err)
	}
	return repo
}

// GetAmendmentEventRepository returns the module's default AmendmentEventRepository
func (k *Keeper) GetAmendmentEventRepository(ctx sdk.Context) *registration.AmendmentEventRepository {
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
func (k *Keeper) GetCommitmentService(ctx sdk.Context) *commitment.CommitmentService {
	return commitment.NewCommitmentService(
		k.GetACPEngine(ctx),
		k.GetRegistrationsCommitmentRepository(ctx),
	)
}

// GetRegistrationService returns the module's default RegistrationService instance
func (k *Keeper) GetRegistrationService(ctx sdk.Context) *registration.RegistrationService {
	return registration.NewRegistrationService(
		k.GetACPEngine(ctx),
		k.GetAmendmentEventRepository(ctx),
		k.GetCommitmentService(ctx),
	)
}

// GetPolicyCmdHandler returns the module's default PolicyCmd Handler instance
func (k *Keeper) GetPolicyCmdHandler(ctx sdk.Context) *policy_cmd.Handler {
	return policy_cmd.NewPolicyCmdHandler(
		k.GetACPEngine(ctx),
		k.GetRegistrationService(ctx),
		k.GetCommitmentService(ctx),
	)
}
