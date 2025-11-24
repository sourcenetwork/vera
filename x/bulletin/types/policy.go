package types

// BasePolicy defines base policy for the bulletin module namespaces.
func BasePolicy() string {
	policyStr := `
description: Base policy that defines permissions for bulletin namespaces
name: Bulletin Policy
resources:
- name: namespace
  permissions:
  - expr: owner + collaborator
    name: create_post
  relations:
  - name: collaborator
    types:
    - actor
  - name: owner
    types:
    - actor
`
	return policyStr
}
