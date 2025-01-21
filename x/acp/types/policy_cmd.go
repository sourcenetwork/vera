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
