package types

import "fmt"

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:              DefaultParams(),
		Rings:               []Ring{},
		Documents:           []Document{},
		KeyDerivations:      []KeyDerivation{},
		NodeDemerits:        []NodeDemeritEntry{},
		AcceptedReportPairs: []AcceptedReportPairEntry{},
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	seenDemerits := map[struct {
		ringID  string
		nodeKey string
	}]struct{}{}
	for _, entry := range gs.NodeDemerits {
		switch {
		case entry.RingId == "":
			return fmt.Errorf("node_demerits ring_id cannot be empty")
		case entry.NodeKey == "":
			return fmt.Errorf("node_demerits node_key cannot be empty")
		case entry.Points == 0:
			return fmt.Errorf("node_demerits points must be at least 1")
		}

		key := struct {
			ringID  string
			nodeKey string
		}{ringID: entry.RingId, nodeKey: entry.NodeKey}
		if _, found := seenDemerits[key]; found {
			return fmt.Errorf("duplicate node_demerits entry for ring %q node %q", entry.RingId, entry.NodeKey)
		}
		seenDemerits[key] = struct{}{}
	}

	seenReportPairs := map[string]struct{}{}
	seenSessionPairs := map[string]struct{}{}
	for _, entry := range gs.AcceptedReportPairs {
		switch {
		case entry.ReportId == "":
			return fmt.Errorf("accepted_report_pairs report_id cannot be empty")
		case entry.SessionId == "":
			return fmt.Errorf("accepted_report_pairs session_id cannot be empty")
		case entry.ExpiresAt == 0:
			return fmt.Errorf("accepted_report_pairs expires_at cannot be zero")
		}
		if _, found := seenReportPairs[entry.ReportId]; found {
			return fmt.Errorf("duplicate accepted_report_pairs entry for report_id %q", entry.ReportId)
		}
		seenReportPairs[entry.ReportId] = struct{}{}
		if _, found := seenSessionPairs[entry.SessionId]; found {
			return fmt.Errorf("duplicate accepted_report_pairs entry for session_id %q", entry.SessionId)
		}
		seenSessionPairs[entry.SessionId] = struct{}{}
	}

	return nil
}
