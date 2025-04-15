package capability

import "github.com/sourcenetwork/acp_core/pkg/errors"

var ErrInvalidCapability error = errors.New("invalid capability", errors.ErrorType_UNAUTHORIZED)
