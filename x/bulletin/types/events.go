package types

const (
	EventRegisterNamespace  = "RegisterNamespace"
	EventAddCollaborator    = "AddCollaborator"
	EventRemoveCollaborator = "RemoveCollaborator"
	EventCreatePost         = "CreatePost"

	AttributeKeyNamespaceId     = "namespace_id"
	AttributeKeyOwnerDid        = "owner_did"
	AttributeKeyCreatedAt       = "created_at"
	AttributeKeyCollaboratorDid = "collaborator_did"
	AttributeKeyAddedBy         = "added_by"
	AttributeKeyRemovedBy       = "removed_by"
	AttributeKeyPostId          = "post_id"
	AttributeKeyCreatorDid      = "creator_did"
	AttributeKeyPayload         = "payload"
	AttributeKeyProof           = "proof"
)
