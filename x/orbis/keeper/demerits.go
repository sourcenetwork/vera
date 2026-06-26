package keeper

import "github.com/sourcenetwork/sourcehub/x/orbis/types"

// DemeritAmountForReportType returns the number of demerit points for a report type.
func DemeritAmountForReportType(ring *types.Ring, reportType string) uint64 {
	if ring == nil {
		return 0
	}

	switch reportType {
	case NodeOfflineReportType:
		return ring.DemeritConfig.NodeOfflineDemerits
	default:
		return 0
	}
}
