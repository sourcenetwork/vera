package ante

import (
	"testing"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/stretchr/testify/require"

	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
)

func TestRejectLegacyTxDecorator_RejectsLegacyAmino(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewRejectLegacyTxDecorator()
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "name: test policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// Create transaction with SIGN_MODE_LEGACY_AMINO_JSON
	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON)
	require.NoError(t, err)

	// The decorator should reject the transaction
	_, err = antehandler(s.ctx, tx, false)
	require.Error(t, err, "RejectLegacyTxDecorator should reject transactions with SIGN_MODE_LEGACY_AMINO_JSON")
	require.Contains(t, err.Error(), "SIGN_MODE_LEGACY_AMINO_JSON")
	require.Contains(t, err.Error(), "not supported")
}

func TestRejectLegacyTxDecorator_AllowsDirectSignMode(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewRejectLegacyTxDecorator()
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(1)

	msg := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "name: test policy",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
	require.NoError(t, s.txBuilder.SetMsgs(msg))

	// Create transaction with SIGN_MODE_DIRECT
	privs, accNums, accSeqs := []cryptotypes.PrivKey{accs[0].priv}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	// The decorator should allow the transaction
	_, err = antehandler(s.ctx, tx, false)
	require.NoError(t, err, "RejectLegacyTxDecorator should allow transactions with SIGN_MODE_DIRECT")
}

func TestRejectLegacyTxDecorator_MultipleSigners(t *testing.T) {
	s := SetupTestSuite(t, true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	decorator := NewRejectLegacyTxDecorator()
	antehandler := sdk.ChainAnteDecorators(decorator)

	accs := s.CreateTestAccounts(2)

	msg1 := &acptypes.MsgCreatePolicy{
		Creator:     accs[0].acc.GetAddress().String(),
		Policy:      "name: test policy 1",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
	msg2 := &acptypes.MsgCreatePolicy{
		Creator:     accs[1].acc.GetAddress().String(),
		Policy:      "name: test policy 2",
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}
	require.NoError(t, s.txBuilder.SetMsgs(msg1, msg2))

	// Create transaction with multiple signers, one using legacy amino
	privs := []cryptotypes.PrivKey{accs[0].priv, accs[1].priv}
	accNums := []uint64{0, 1}
	accSeqs := []uint64{0, 0}

	tx, err := s.CreateTestTx(s.ctx, privs, accNums, accSeqs, s.ctx.ChainID(), signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON)
	require.NoError(t, err)

	// The decorator should reject the transaction
	_, err = antehandler(s.ctx, tx, false)
	require.Error(t, err, "RejectLegacyTxDecorator should reject transactions where any signer uses SIGN_MODE_LEGACY_AMINO_JSON")
	require.Contains(t, err.Error(), "SIGN_MODE_LEGACY_AMINO_JSON")
}
