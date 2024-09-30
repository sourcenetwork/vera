package registration

import "github.com/sourcenetwork/sourcehub/x/acp/types"

// verifies a merkle commitment thingy
// generate a commitment from a set of object ids
// utility to generate an opening from a batched commitment

func VerifyProof(commitment []byte, proof *types.RegistrationProof) bool {
	return false
}
