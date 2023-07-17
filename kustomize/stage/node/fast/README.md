# Node Fast Stage

This Stage only initializes the node.

The `node-initialize` Stage is applied to nodes that do not have any conditions set in their `status.conditions` field.
When applied, this Stage sets the `status.conditions` field for the node, as well as the `status.addresses`, `status.allocatable`,
and `status.capacity` fields.
