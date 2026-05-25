package types

// BasePolicy defines the base module-level policy for orbis namespaces.
// `member` is the relation granting create_ring permission on a namespace.
func BasePolicy() string {
	return `
name: Orbis Policy
description: Base policy that defines permissions for orbis namespaces
resources:
- name: namespace
  permissions:
  - expr: member
    name: create_ring
  relations:
  - name: member
    types:
    - actor
`
}
