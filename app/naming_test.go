package app

import (
	"path/filepath"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	coremodulev1 "github.com/sourcenetwork/vera/api/vera/core/module"
	acptypes "github.com/sourcenetwork/vera/x/acp/types"
	coretypes "github.com/sourcenetwork/vera/x/core/types"
)

func TestVeraIdentity(t *testing.T) {
	require.Equal(t, "vera", Name)
	require.Equal(t, ".vera", filepath.Base(DefaultNodeHome))
	require.Equal(t, "/vera.acp.MsgCreatePolicy", sdk.MsgTypeURL(&acptypes.MsgCreatePolicy{}))
	require.Equal(t, "core", coretypes.ModuleName)
	require.Equal(t, "vera.core.module.Module", string((&coremodulev1.Module{}).ProtoReflect().Descriptor().FullName()))
	require.Equal(t, "/vera.core.MsgUpdateParams", sdk.MsgTypeURL(&coretypes.MsgUpdateParams{}))
}
