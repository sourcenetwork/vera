package types

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"

	prototypes "github.com/cosmos/gogoproto/types"

	"github.com/sourcenetwork/acp_core/pkg/types"
	"github.com/sourcenetwork/sourcehub/utils"
)

// ProduceId uses all fields in an AccessDecision (ignoring the ID) to produce an ID
// for the Decision.
func (d *AccessDecision) ProduceId() string {
	hash := d.hashDecision()
	return base32.StdEncoding.EncodeToString(hash)
}

// hashDecision produces a sha256 hash of all fields (except the id) of an AccessDecision.
// The hash is used to produce an unique and deterministic ID for a decision
func (decision *AccessDecision) hashDecision() []byte {
	sortableOperations := utils.FromComparator(decision.Operations, func(left, right *types.Operation) bool {
		return left.Object.Resource < right.Object.Resource && left.Object.Id < right.Object.Id && left.Permission < right.Permission
	})
	operations := sortableOperations.Sort()

	hasher := sha256.New()
	hasher.Write([]byte(decision.PolicyId))
	hasher.Write([]byte(decision.Creator))
	hasher.Write([]byte(decision.Actor))
	hasher.Write([]byte(fmt.Sprintf("%v", decision.CreatorAccSequence)))
	hasher.Write([]byte(fmt.Sprintf("%v", decision.IssuedHeight)))
	hasher.Write([]byte(prototypes.TimestampString(decision.CreationTime.ProtoTs)))

	for _, operation := range operations {
		hasher.Write([]byte(operation.Object.Resource))
		hasher.Write([]byte(operation.Object.Id))
		hasher.Write([]byte(operation.Permission))
	}

	hasher.Write(decision.hashParams())

	return hasher.Sum(nil)
}

// hashParams produces a sha256 of the DecisionParameters
func (d *AccessDecision) hashParams() []byte {
	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%v", d.Params.DecisionExpirationDelta)))
	hasher.Write([]byte(fmt.Sprintf("%v", d.Params.ProofExpirationDelta)))
	hasher.Write([]byte(fmt.Sprintf("%v", d.Params.TicketExpirationDelta)))
	return hasher.Sum(nil)
}
