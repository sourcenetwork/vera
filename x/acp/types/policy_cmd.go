package types

import acptypes "github.com/sourcenetwork/acp_core/pkg/types"

func NewSetRelationshipCmd(rel *acptypes.Relationship) *PolicyCmd {
	return &PolicyCmd{
		Cmd: &PolicyCmd_SetRelationshipCmd{
			SetRelationshipCmd: &SetRelationshipCmd{
				Relationship: rel,
			},
		},
	}
}

func NewDeleteRelationshipCmd(rel *acptypes.Relationship) *PolicyCmd {
	return &PolicyCmd{
		Cmd: &PolicyCmd_DeleteRelationshipCmd{
			DeleteRelationshipCmd: &DeleteRelationshipCmd{
				Relationship: rel,
			},
		},
	}
}

func NewRegisterObjectCmd(obj *acptypes.Object) *PolicyCmd {
	return &PolicyCmd{
		Cmd: &PolicyCmd_RegisterObjectCmd{
			RegisterObjectCmd: &RegisterObjectCmd{
				Object: obj,
			},
		},
	}
}

func NewArchiveObjectCmd(obj *acptypes.Object) *PolicyCmd {
	return &PolicyCmd{
		Cmd: &PolicyCmd_ArchiveObjectCmd{
			ArchiveObjectCmd: &ArchiveObjectCmd{
				Object: obj,
			},
		},
	}
}

func NewCommitRegistrationCmd(commitment []byte) *PolicyCmd {
	return &PolicyCmd{
		Cmd: &PolicyCmd_CommitRegistrationsCmd{
			CommitRegistrationsCmd: &CommitRegistrationsCmd{
				Commitment: commitment,
			},
		},
	}
}

func NewRevealRegistrationCmd(commitmentId uint64, proof *RegistrationProof) *PolicyCmd {
	return &PolicyCmd{
		Cmd: &PolicyCmd_RevealRegistrationCmd{
			RevealRegistrationCmd: &RevealRegistrationCmd{
				Proof:                     proof,
				RegistrationsCommitmentId: commitmentId,
			},
		},
	}
}

func NewFlagHijackAttemptCmd(eventId uint64) *PolicyCmd {
	return &PolicyCmd{
		Cmd: &PolicyCmd_FlagHijackAttemptCmd{
			FlagHijackAttemptCmd: &FlagHijackAttemptCmd{
				EventId: eventId,
			},
		},
	}
}

func NewUnarchiveObjectCmd(object *acptypes.Object) *PolicyCmd {
	return &PolicyCmd{
		Cmd: &PolicyCmd_UnarchiveObjectCmd{
			UnarchiveObjectCmd: &UnarchiveObjectCmd{
				Object: object,
			},
		},
	}
}
