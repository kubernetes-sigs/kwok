# Pod General Stage

These Stages simulate real Pod behavior as closely as possible, including various container states,
failures, and restarts.

## Stages

### 1. pod-create.yaml
Creates a pod with Pending status and sets up initial container statuses.

### 2. pod-init-container-running.yaml
Transitions init containers from waiting to running state.

### 3. pod-init-container-completed.yaml
Completes init containers. Properly handles both regular init containers (terminated state) 
and sidecar init containers with `restartPolicy: Always` (running state).

### 4. pod-init-container-failed.yaml
Simulates init container failure. Triggered by annotation:
- `pod-init-container-failed.stage.kwok.x-k8s.io/name`: name of the failing init container
- `pod-init-container-failed.stage.kwok.x-k8s.io/exit-code`: exit code (default: 1)
- `pod-init-container-failed.stage.kwok.x-k8s.io/reason`: failure reason (default: Error)
- `pod-init-container-failed.stage.kwok.x-k8s.io/message`: failure message

### 5. pod-init-container-restart.yaml
Restarts a failed init container, incrementing the restart count. Automatically triggers
after init container failure.

### 6. pod-ready.yaml
Transitions pod to Running phase with all containers running. Properly preserves 
init container statuses including sidecars.

### 7. pod-container-failed.yaml
Simulates regular container failure. Triggered by annotation:
- `pod-container-failed.stage.kwok.x-k8s.io/name`: name of the failing container
- `pod-container-failed.stage.kwok.x-k8s.io/exit-code`: exit code (default: 1)
- `pod-container-failed.stage.kwok.x-k8s.io/reason`: failure reason (default: Error)
- `pod-container-failed.stage.kwok.x-k8s.io/message`: failure message

### 8. pod-container-restart.yaml
Restarts a failed container, incrementing the restart count. Automatically triggers
after container failure.

### 9. pod-complete.yaml
Completes pods owned by Jobs with Succeeded status.

### 10. pod-delete.yaml
Deletes pods that have a deletion timestamp set.

## Supported Behaviors

### InitContainer Behaviors
- ✅ Regular init containers that complete successfully
- ✅ Sidecar init containers (`restartPolicy: Always`) that keep running
- ✅ Init container failures with custom exit codes
- ✅ Init container restart/retry after failure

### Container Behaviors
- ✅ Containers running normally
- ✅ Containers completing (for Jobs)
- ✅ Container failures with custom exit codes and reasons
- ✅ Container restart/retry after failure

### Pod Behaviors
- ✅ Pod creation → Pending
- ✅ Pod → Running (after init containers complete)
- ✅ Pod → Succeeded (for Jobs)
- ✅ Pod deletion
- ✅ Pod failures (result of container failures)

## Usage Examples

### Basic Pod
A pod without init containers will transition: Pending → Running

### Pod with Init Container
A pod with init containers will transition: Pending → Init:Running → Running

### Pod with Sidecar
A pod with sidecar init containers (`restartPolicy: Always`) will keep the sidecar running
even after main containers start.

### Simulating Init Container Failure
Add annotations to trigger init container failure:
```yaml
metadata:
  annotations:
    pod-init-container-failed.stage.kwok.x-k8s.io/name: init-container-name
    pod-init-container-failed.stage.kwok.x-k8s.io/exit-code: "1"
```

The init container will fail and then automatically restart.

### Simulating Container Failure
Add annotations to trigger container failure:
```yaml
metadata:
  annotations:
    pod-container-failed.stage.kwok.x-k8s.io/name: container-name
    pod-container-failed.stage.kwok.x-k8s.io/exit-code: "137"
    pod-container-failed.stage.kwok.x-k8s.io/reason: OOMKilled
```

The container will fail and then automatically restart.

