package object_registration

import "github.com/sourcenetwork/sourcehub/x/acp/types"

// CommitmentExpirationDuration sets the default expiration time for commitments in the test setup
var CommitmentExpirationDuration = types.Duration{
	Duration: &types.Duration_BlockCount{
		BlockCount: 5,
	},
}

// OperationsPerTest models the amount of operations which should be executed every test
var OperationsPerTest = 20

var TestCount = 10

var ActorCount = 3

var MaxTriesPerTest = 40
