package object_registration

const Policy string = `
name: test
resources:
  file:
    relations:
      owner:
        types:
          - actor
`

const ResourceName = "file"
const ObjectId = "readme.txt"
