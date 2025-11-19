package commitment

import (
	"fmt"

	"github.com/sourcenetwork/acp_core/pkg/errors"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func errInvalidCommitment(policy string, commitment []byte) error {
	return types.New(fmt.Sprintf("invalid commitment size : got %v, want %v bytes", len(commitment), commitmentBytes),
		types.ErrorType_BAD_INPUT,
		errors.Pair("policy", policy))
}
