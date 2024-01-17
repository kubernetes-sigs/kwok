# KEP-922: v1alpha2 API

<!-- toc -->
- [KEP-922: v1alpha2 API](#kep-922-v1alpha2-api)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Design Details](#design-details)
    - [Stage](#stage)
      - [Patch Template Changes](#patch-template-changes)
      - [Delay Changes](#delay-changes)
      - [Match Expressions Changes](#match-expressions-changes)
    - [PodShadow](#podshadow)
    - [Test Plan](#test-plan)
        - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit Tests](#unit-tests)
      - [Integration tests](#integration-tests)
    - [Graduation Criteria](#graduation-criteria)
  - [Implementation History](#implementation-history)
  - [Drawbacks](#drawbacks)
  - [Alternatives](#alternatives)
<!-- /toc -->


## Summary

This KEP proposes a migration path for the `v1alpha1` API to the new `v1alpha2` API,
introducing several breaking changes to the `Stage` resource's patch templates, delay/jitter duration configuration, and match expressions in selectors.
Additionally, we're introducing a resource named `PodShadow`, amalgamating existing resources such as `ClusterExec`, `Exec`, `ClusterPortForward`, `PortForward`, `ClusterLogs`, `Logs`, `ClusterAttach`, and `Attach`.
This transition aims to make the protocol more intuitive and user-friendly, with clearly defined use-cases.

## Motivation

The changes in the `v1alpha2` API will provide more fine-grained control over the desired state and more flexibility.

### Goals

1. Outline the changes between `v1alpha1` and `v2alpha2`.
2. Automate conversion `v1alpha1` to `v2alpha2`.

### Non-Goals

This KEP does not aim to alter or modify the primary functionalities of the current resources,
it is not to change the core concepts but to streamline them and elevate user interaction.

## Proposal

The `v1alpha2` API proposes significant changes in the APIs' structural layout,
focusing on patch templates, delay, and match expressions. It also introduces a unified resource known as PodShadow,
aiming to streamline user workflows.

### Risks and Mitigations

The key risk is disruption of existing configurations and deployments using the `v1alpha1` API when migrated to `v1alpha2`.
Rigorous testing, careful planning, and phased transitions can mitigate this risk.
Furthermore, confusion or misunderstanding about the new API structure could lead to deployment issues and misconfigurations.
Clear documentation, examples, and guides should be provided to limit this risk.

## Design Details

### Stage

#### Patch Template Changes

Old Structure:
```yaml
spec:
  next:
    statusTemplate: <string>     # The status template.
    statusSubresource: <string>  # The status subresource. default: 'status'
    statusPatchAs:
      username: <string>         # The username to impersonate.
```

New Structure:
```yaml
spec:
  next:
    patches:
    - subresource: <string>  # The subresource to patch. This could be '', 'status', 'spec', 'scale' etc. the '' is used to patch the entire resource. default: ''
      type: <string>         # The type of the patch. This could be 'merge', 'strategic' etc.
      root: <string>         # The root of the template calculation. This could be '', 'status', 'spec', etc. the '' is used to patch the entire resource. default: ''
      template: <string>     # The template to apply. This is a string representation of the desired state.
      as:                    # The as section is used to impersonate a user or group.
        username: <string>   # User could be a regular user or a service account in a namespace.
        groups:
        - <string>
```

Examples:

Update a replicas through the 'scale' subresource

```yaml
spec:
  next:
    patches:
    - subresource: scale
      type: merge
      root: spec
      template: |
        replicas: 2
```

Update the 'status' but not is a subresource

```yaml
spec:
  next:
    patches:
    - subresource: ''
      type: merge
      root: status
      template: |
        phase: Running
```

#### Delay Changes

Old Structure:
```yaml
spec:
  delay:
    durationMilliseconds: <int>  # The delay duration in milliseconds.
    durationFrom:
      expressionFrom: <expressions-string>  # The expression from which the delay duration is derived.
    jitterDurationMilliseconds: <int>  # The jitter duration in milliseconds.
    jitterDurationFrom:
      expressionFrom: <expressions-string>  # The expression from which the jitter duration is derived.
```

New Structure:
```yaml
spec:
  delay:
    duration:
      milliseconds: <int>  # The delay duration in milliseconds. This is a positive integer.
      jq:
        expression: <jq-expressions-string>  # The jq expression from which the delay duration is derived. This is a string representation of a jq expression.
      cel:
        expression: <cel-expressions-string>  # The cel expression from which the delay duration is derived. This is a string representation of a cel expression.
    jitterDuration:
      milliseconds: <int>  # The jitter duration in milliseconds. This is a positive integer.
      jq:
        expression: <jq-expressions-string>  # The jq expression from which the jitter duration is derived. This is a string representation of a jq expression.
      cel:
        expression: <cel-expressions-string>  # The cel expression from which the jitter duration is derived. This is a string representation of a cel expression.
```

#### Match Expressions Changes

Old Structure:
```yaml
spec:
  selector:
    matchExpressions:
    - key: <expressions-string>  # The key for the match expression.
      operator: <string>  # The operator for the match expression. This could be 'In', 'NotIn', etc.
      values:
      - <string>  # The values for the match expression. This is a list of strings.
```

New Structure:
```yaml
spec:
  selector:
    jq:
    - key: <jq-expressions-string>  # The key for the jq match expression. This is a string representation of a jq expression.
      operator: <string>  # The operator for the jq match expression. This could be 'In', 'NotIn', etc.
      values:
      - <string>  # The values for the jq match expression. This is a list of strings.
    cel:
    - expression: <cel-expressions-string>  # The cel match expression. This is a string representation of one or more cel expressions.
```

### PodShadow

Merge the `ClusterExec`, `Exec`, `ClusterPortForward`, `PortForward`, `ClusterLogs`, `Logs`, `ClusterAttach`, and `Attach` into a single unified resource named `PodShadow`.

> Q: Why name it `PodShadow`?
> A: `PodShadow` is inspired by the ghost in the KWOK logo.
> Ghosts, as per folklore, lack shadows, hence naming it `PodShadow` gives our ghostly pod entity.

```yaml
kind: PodShadow
apiVersion: kwok.x-k8s.io/v1alpha2
metadata:
  name: rules
spec:
  selector:
    matchNamespaces:
    - <namespace>
    matchNames:
    - <pod-name>
    matchContainerNames:
    - <container-name>
    matchContainerPorts:
    - <container-port>

  exec:
    local:
      workDir: <string>
      envs:
      - name: <string>
        value: <string> 
 
  portForward:
    target:
      port: <int>
      address: <string>
    command:
    - <string>
    - <string>

  logs:
    logsFile: <string>
    follow: <bool>

  attach:
    logsFile: <string>
```

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

#### Unit Tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- `<package>`: `<date>` - `<test coverage>`

#### Integration tests

<!--
Describe what tests will be added to ensure proper quality of the enhancement.

After the implementation PR is merged, add the names of the tests here.
-->

### Graduation Criteria

<!--

Clearly define what it means for the feature to be implemented and
considered stable.

If the feature you are introducing has high complexity, consider adding graduation
milestones with these graduation criteria:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/
-->

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->
