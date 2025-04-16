package types

// BasePolicy defines base policy for the bulletin module namespaces.
func BasePolicy() string {
	policyStr := `
name: Bulletin Policy
description: Base policy that defines permissions for bulletin namespaces
resources:
  namespace:
    relations:
      owner:
        types:
          - actor
      collaborator:
        types: 
          - actor
    permissions:
      create_post:
        expr: owner + collaborator
`
	return policyStr
}
