# Pod Fast Stage

These Stages make the pod ready, completed or deleted.

The `pod-ready` Stage is applied to pods that do not have a `status.podIP` set and do not have a `metadata.deletionTimestamp` set.
When applied, this Stage sets the `status.conditions`, `status.containerStatuses`, and `status.initContainerStatuses` fields for the pod,
as well as the `status.hostIP` and `status.podIP` fields. It will also set the phase and startTime fields, indicating that the pod is running and has been started.

The `pod-complete` Stage is applied to pods that are running, do not have a `metadata.deletionTimestamp` set,
and are owned by a Job. When applied, this Stage updates the `status.containerStatuses` field for the pod,
setting the ready and started fields to true and the `state.terminated` field to indicate that the pod has completed.
It also sets the phase field to Succeeded, indicating that the pod has completed successfully.

The `pod-delete` Stage is applied to pods that have a `metadata.deletionTimestamp` set.
When applied, this Stage empties the `metadata.finalizers` field for the pod, allowing it to be deleted, and then delete the pod.
