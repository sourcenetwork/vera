package object_registration

const ResourceName = "file"
const ObjectId = "readme.txt"
const Policy string = `
name: test
resources:
  file:
    relations:
      owner:
        types:
          - actor
`
