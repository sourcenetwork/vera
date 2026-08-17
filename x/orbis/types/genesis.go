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

	knownRings := make(map[string]struct{}, len(gs.Rings))
	for _, r := range gs.Rings {
		knownRings[r.Id] = struct{}{}
		if err := ValidateTrustedAuthRelayConfig(r.AllowTrustedAuthRelays, r.TrustedAuthRelayDids); err != nil {
			return fmt.Errorf("ring %q trusted auth relays: %w", r.Id, err)
		}

		// A zero-value reporting config means "use module defaults" and is
		// backfilled from params during InitGenesis.
		if r.Reporting.Equal(ReportingConfig{}) {
			continue
		}
		if err := ValidateReportingConfig(r.Reporting); err != nil {
			return fmt.Errorf("ring %q reporting config: %w", r.Id, err)
		}
		seenBackups := make(map[string]struct{}, len(r.Reporting.BackupNodeKeys))
		for _, nodeKey := range r.Reporting.BackupNodeKeys {
			if _, found := seenBackups[nodeKey]; found {
				return fmt.Errorf("ring %q reporting config: duplicate backup node key %q", r.Id, nodeKey)
			}
			seenBackups[nodeKey] = struct{}{}
		}
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
		if _, ok := knownRings[entry.RingId]; !ok {
			return fmt.Errorf("node_demerits references unknown ring %q", entry.RingId)
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
