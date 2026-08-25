package keeper

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/sourcenetwork/vera/x/orbis/types"
)

// DemeritAmountForReportType returns the number of demerit points for a report type.
func DemeritAmountForReportType(ring *types.Ring, reportType string) (uint64, error) {
	if ring == nil {
		return 0, types.ErrRingNotFound
	}

	switch reportType {
	case NodeOfflineReportType:
		return ring.Reporting.DemeritConfig.NodeOfflineDemerits, nil
	case InvalidCryptoResponseReportType:
		return ring.Reporting.DemeritConfig.InvalidCryptoResponseDemerits, nil
	case UnauthorizedRequestReportType:
		return ring.Reporting.DemeritConfig.UnauthorizedRequestDemerits, nil
	default:
		return 0, errorsmod.Wrapf(types.ErrInvalidReport, "unsupported report type %q", reportType)
	}
}
