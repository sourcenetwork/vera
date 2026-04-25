package types

// BasePolicy defines base policy for the bulletin module namespaces.
// `collaborator` is the single relation used for both posting and editing.
// `update_post` is gated by the `collaborator` relation. The post creator
// is always allowed to update their own post regardless of this policy
// (checked directly in the keeper).
func BasePolicy() string {
	policyStr := `
description: Base policy that defines permissions for bulletin namespaces
name: Bulletin Policy
resources:
- name: namespace
  permissions:
  - expr: collaborator
    name: create_post
  - expr: collaborator
    name: update_post
  relations:
  - name: collaborator
    types:
    - actor
`

	return policyStr
}
