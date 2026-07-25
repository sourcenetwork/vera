package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestLockupKeyRoundTripWithSeparatorBytes(t *testing.T) {
	delegator := addressWithSeparator(3)
	validator := sdk.ValAddress(addressWithSeparator(11))

	gotDelegator, gotValidator := LockupKeyToAddresses(LockupKey(delegator, validator))

	require.Equal(t, delegator, gotDelegator)
	require.Equal(t, validator, gotValidator)
}

func TestUnlockingLockupKeyRoundTripWithSeparatorBytes(t *testing.T) {
	delegator := addressWithSeparator(7)
	validator := sdk.ValAddress(addressWithSeparator(15))

	gotDelegator, gotValidator, gotHeight := UnlockingLockupKeyToAddressesAtHeight(
		UnlockingLockupKey(delegator, validator, 12345),
	)

	require.Equal(t, delegator, gotDelegator)
	require.Equal(t, validator, gotValidator)
	require.Equal(t, int64(12345), gotHeight)
}

func TestUserSubscriptionKeyRoundTripWithSeparators(t *testing.T) {
	developer := addressWithSeparator(5)
	userDID := "did:web:example.com/users/alice"

	gotDeveloper, gotUserDID := UserSubscriptionKeyToAddresses(
		UserSubscriptionKey(developer, userDID),
	)

	require.Equal(t, developer, gotDeveloper)
	require.Equal(t, userDID, gotUserDID)
}

func TestDeveloperKeyRoundTripWithSeparatorByte(t *testing.T) {
	developer := addressWithSeparator(9)

	require.Equal(t, developer, DeveloperKeyToAddress(DeveloperKey(developer)))
}

func addressWithSeparator(index int) sdk.AccAddress {
	address := make(sdk.AccAddress, addressKeyLength)
	for i := range address {
		address[i] = byte(i + 1)
	}
	address[index] = keySeparator
	return address
}
