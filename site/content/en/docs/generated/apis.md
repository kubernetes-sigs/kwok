---
title: API reference
bookToc: false
---
<h1>API reference</h1>
<p>Packages:</p>
<ul>
<li>
<a href="#action.kwok.x-k8s.io%2fv1alpha1">action.kwok.x-k8s.io/v1alpha1</a>
</li>
<li>
<a href="#config.kwok.x-k8s.io%2fv1alpha1">config.kwok.x-k8s.io/v1alpha1</a>
</li>
<li>
<a href="#kwok.x-k8s.io%2fv1alpha1">kwok.x-k8s.io/v1alpha1</a>
</li>
</ul>
<h2 id="action.kwok.x-k8s.io/v1alpha1">
action.kwok.x-k8s.io/v1alpha1
<a href="#action.kwok.x-k8s.io%2fv1alpha1"> #</a>
</h2>
<div>
<p>Package v1alpha1 implements the v1alpha1 apiVersion of kwokctl&rsquo;s action</p>
</div>
Resource Types:
<ul>
<li>
<a href="#action.kwok.x-k8s.io/v1alpha1.ResourcePatch">ResourcePatch</a>
</li></ul>
<h3 id="action.kwok.x-k8s.io/v1alpha1.ResourcePatch">
ResourcePatch
<a href="#action.kwok.x-k8s.io%2fv1alpha1.ResourcePatch"> #</a>
</h3>
<p>
<p>ResourcePatch provides resource definition for kwokctl.
this is a action of resource patch.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code>
string
</td>
<td>
<code>
action.kwok.x-k8s.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code>
string
</td>
<td><code>ResourcePatch</code></td>
</tr>
<tr>
<td>
<code>resource</code>
<em>
<a href="#action.kwok.x-k8s.io/v1alpha1.GroupVersionResource">
GroupVersionResource
</a>
</em>
</td>
<td>
<p>Resource represents the resource to be patched.</p>
</td>
</tr>
<tr>
<td>
<code>target</code>
<em>
<a href="#action.kwok.x-k8s.io/v1alpha1.Target">
Target
</a>
</em>
</td>
<td>
<p>Target represents the target of the ResourcePatch.</p>
</td>
</tr>
<tr>
<td>
<code>durationNanosecond</code>
<em>
time.Duration
</em>
</td>
<td>
<p>DurationNanosecond represents the duration of the patch in nanoseconds.</p>
</td>
</tr>
<tr>
<td>
<code>method</code>
<em>
<a href="#action.kwok.x-k8s.io/v1alpha1.PatchMethod">
PatchMethod
</a>
</em>
</td>
<td>
<p>Method represents the method of the patch.</p>
</td>
</tr>
<tr>
<td>
<code>template</code>
<em>
encoding/json.RawMessage
</em>
</td>
<td>
<p>Template contains the patch data as a raw JSON message.</p>
</td>
</tr>
</tbody>
</table>
<h2 id="config.kwok.x-k8s.io/v1alpha1">
config.kwok.x-k8s.io/v1alpha1
<a href="#config.kwok.x-k8s.io%2fv1alpha1"> #</a>
</h2>
<div>
<p>Package v1alpha1 implements the v1alpha1 apiVersion of kwok&rsquo;s configuration</p>
</div>
Resource Types:
<ul>
<li>
<a href="#config.kwok.x-k8s.io/v1alpha1.KwokConfiguration">KwokConfiguration</a>
</li>
<li>
<a href="#config.kwok.x-k8s.io/v1alpha1.KwokctlComponent">KwokctlComponent</a>
</li>
<li>
<a href="#config.kwok.x-k8s.io/v1alpha1.KwokctlConfiguration">KwokctlConfiguration</a>
</li>
<li>
<a href="#config.kwok.x-k8s.io/v1alpha1.KwokctlResource">KwokctlResource</a>
</li></ul>
<h3 id="config.kwok.x-k8s.io/v1alpha1.KwokConfiguration">
KwokConfiguration
<a href="#config.kwok.x-k8s.io%2fv1alpha1.KwokConfiguration"> #</a>
</h3>
<p>
<p>KwokConfiguration provides configuration for the Kwok.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code>
string
</td>
<td>
<code>
config.kwok.x-k8s.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code>
string
</td>
<td><code>KwokConfiguration</code></td>
</tr>
<tr>
<td>
<code>metadata</code>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<p>Standard list metadata.
More info: <a href="https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata">https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata</a></p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>options</code>
<em>
<a href="#config.kwok.x-k8s.io/v1alpha1.KwokConfigurationOptions">
KwokConfigurationOptions
</a>
</em>
</td>
<td>
<p>Options holds information about the default value.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.kwok.x-k8s.io/v1alpha1.KwokctlComponent">
KwokctlComponent
<a href="#config.kwok.x-k8s.io%2fv1alpha1.KwokctlComponent"> #</a>
</h3>
<p>
<p>KwokctlComponent  holds information about the kwokctl component.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code>
string
</td>
<td>
<code>
config.kwok.x-k8s.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code>
string
</td>
<td><code>KwokctlComponent</code></td>
</tr>
<tr>
<td>
<code>metadata</code>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<p>Standard list metadata.
More info: <a href="https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata">https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata</a></p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>parameters</code>
<em>
encoding/json.RawMessage
</em>
</td>
<td>
<p>Parameters is the parameters for the kwokctl component configuration.</p>
</td>
</tr>
<tr>
<td>
<code>template</code>
<em>
string
</em>
</td>
<td>
<p>Template is the template for the kwokctl component configuration.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.kwok.x-k8s.io/v1alpha1.KwokctlConfiguration">
KwokctlConfiguration
<a href="#config.kwok.x-k8s.io%2fv1alpha1.KwokctlConfiguration"> #</a>
</h3>
<p>
<p>KwokctlConfiguration provides configuration for the Kwokctl.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code>
string
</td>
<td>
<code>
config.kwok.x-k8s.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code>
string
</td>
<td><code>KwokctlConfiguration</code></td>
</tr>
<tr>
<td>
<code>metadata</code>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<p>Standard list metadata.
More info: <a href="https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata">https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata</a></p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>options</code>
<em>
<a href="#config.kwok.x-k8s.io/v1alpha1.KwokctlConfigurationOptions">
KwokctlConfigurationOptions
</a>
</em>
</td>
<td>
<p>Options holds information about the default value.</p>
</td>
</tr>
<tr>
<td>
<code>components</code>
<em>
<a href="#config.kwok.x-k8s.io/v1alpha1.Component">
[]Component
</a>
</em>
</td>
<td>
<p>Components holds information about the components.</p>
</td>
</tr>
<tr>
<td>
<code>componentsPatches</code>
<em>
<a href="#config.kwok.x-k8s.io/v1alpha1.ComponentPatches">
[]ComponentPatches
</a>
</em>
</td>
<td>
<p>ComponentsPatches holds information about the components patches.</p>
</td>
</tr>
<tr>
<td>
<code>status</code>
<em>
<a href="#config.kwok.x-k8s.io/v1alpha1.KwokctlConfigurationStatus">
KwokctlConfigurationStatus
</a>
</em>
</td>
<td>
<p>Status holds information about the status.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.kwok.x-k8s.io/v1alpha1.KwokctlResource">
KwokctlResource
<a href="#config.kwok.x-k8s.io%2fv1alpha1.KwokctlResource"> #</a>
</h3>
<p>
<p>KwokctlResource provides resource definition for kwokctl.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code>
string
</td>
<td>
<code>
config.kwok.x-k8s.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code>
string
</td>
<td><code>KwokctlResource</code></td>
</tr>
<tr>
<td>
<code>metadata</code>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<p>Standard list metadata.
More info: <a href="https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata">https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata</a></p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>parameters</code>
<em>
encoding/json.RawMessage
</em>
</td>
<td>
<p>Parameters is the parameters for the kwokctl resource configuration.</p>
</td>
</tr>
<tr>
<td>
<code>template</code>
<em>
string
</em>
</td>
<td>
<p>Template is the template for the kwokctl resource configuration.</p>
</td>
</tr>
</tbody>
</table>
<h2 id="kwok.x-k8s.io/v1alpha1">
kwok.x-k8s.io/v1alpha1
<a href="#kwok.x-k8s.io%2fv1alpha1"> #</a>
</h2>
<div>
<p>Package v1alpha1 implements the v1alpha1 apiVersion of kwok&rsquo;s configuration</p>
</div>
Resource Types:
<ul>
<li>
<a href="#kwok.x-k8s.io/v1alpha1.Attach">Attach</a>
</li>
<li>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterAttach">ClusterAttach</a>
</li>
<li>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterExec">ClusterExec</a>
</li>
<li>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterLogs">ClusterLogs</a>
</li>
<li>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterPortForward">ClusterPortForward</a>
</li>
<li>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterResourceUsage">ClusterResourceUsage</a>
</li>
<li>
<a href="#kwok.x-k8s.io/v1alpha1.Exec">Exec</a>
</li>
<li>
<a href="#kwok.x-k8s.io/v1alpha1.Logs">Logs</a>
</li>
<li>
<a href="#kwok.x-k8s.io/v1alpha1.Metric">Metric</a>
</li>
<li>
<a href="#kwok.x-k8s.io/v1alpha1.PortForward">PortForward</a>
</li>
<li>
<a href="#kwok.x-k8s.io/v1alpha1.ResourceUsage">ResourceUsage</a>
</li>
<li>
<a href="#kwok.x-k8s.io/v1alpha1.Stage">Stage</a>
</li></ul>
<h3 id="kwok.x-k8s.io/v1alpha1.Attach">
Attach
<a href="#kwok.x-k8s.io%2fv1alpha1.Attach"> #</a>
</h3>
<p>
<p>Attach provides attach configuration for a single pod.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code>
string
</td>
<td>
<code>
kwok.x-k8s.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code>
string
</td>
<td><code>Attach</code></td>
</tr>
<tr>
<td>
<code>metadata</code>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<p>Standard list metadata.
More info: <a href="https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata">https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata</a></p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.AttachSpec">
AttachSpec
</a>
</em>
</td>
<td>
<p>Spec holds spec for attach</p>
<table>
<tr>
<td>
<code>attaches</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.AttachConfig">
[]AttachConfig
</a>
</em>
</td>
<td>
<p>Attaches is a list of attaches to configure.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.AttachStatus">
AttachStatus
</a>
</em>
</td>
<td>
<p>Status holds status for attach</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ClusterAttach">
ClusterAttach
<a href="#kwok.x-k8s.io%2fv1alpha1.ClusterAttach"> #</a>
</h3>
<p>
<p>ClusterAttach provides cluster-wide logging configuration</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code>
string
</td>
<td>
<code>
kwok.x-k8s.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code>
string
</td>
<td><code>ClusterAttach</code></td>
</tr>
<tr>
<td>
<code>metadata</code>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<p>Standard list metadata.
More info: <a href="https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata">https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata</a></p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterAttachSpec">
ClusterAttachSpec
</a>
</em>
</td>
<td>
<p>Spec holds spec for cluster attach.</p>
<table>
<tr>
<td>
<code>selector</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ObjectSelector">
ObjectSelector
</a>
</em>
</td>
<td>
<p>Selector is a selector to filter pods to configure.</p>
</td>
</tr>
<tr>
<td>
<code>attaches</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.AttachConfig">
[]AttachConfig
</a>
</em>
</td>
<td>
<p>Attaches is a list of attach configurations.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterAttachStatus">
ClusterAttachStatus
</a>
</em>
</td>
<td>
<p>Status holds status for cluster attach</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ClusterExec">
ClusterExec
<a href="#kwok.x-k8s.io%2fv1alpha1.ClusterExec"> #</a>
</h3>
<p>
<p>ClusterExec provides cluster-wide exec configuration.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code>
string
</td>
<td>
<code>
kwok.x-k8s.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code>
string
</td>
<td><code>ClusterExec</code></td>
</tr>
<tr>
<td>
<code>metadata</code>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<p>Standard list metadata.
More info: <a href="https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata">https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata</a></p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterExecSpec">
ClusterExecSpec
</a>
</em>
</td>
<td>
<p>Spec holds spec for cluster exec.</p>
<table>
<tr>
<td>
<code>selector</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ObjectSelector">
ObjectSelector
</a>
</em>
</td>
<td>
<p>Selector is a selector to filter pods to configure.</p>
</td>
</tr>
<tr>
<td>
<code>execs</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ExecTarget">
[]ExecTarget
</a>
</em>
</td>
<td>
<p>Execs is a list of exec to configure.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterExecStatus">
ClusterExecStatus
</a>
</em>
</td>
<td>
<p>Status holds status for cluster exec</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ClusterLogs">
ClusterLogs
<a href="#kwok.x-k8s.io%2fv1alpha1.ClusterLogs"> #</a>
</h3>
<p>
<p>ClusterLogs provides cluster-wide logging configuration</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code>
string
</td>
<td>
<code>
kwok.x-k8s.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code>
string
</td>
<td><code>ClusterLogs</code></td>
</tr>
<tr>
<td>
<code>metadata</code>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<p>Standard list metadata.
More info: <a href="https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata">https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata</a></p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterLogsSpec">
ClusterLogsSpec
</a>
</em>
</td>
<td>
<p>Spec holds spec for cluster logs.</p>
<table>
<tr>
<td>
<code>selector</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ObjectSelector">
ObjectSelector
</a>
</em>
</td>
<td>
<p>Selector is a selector to filter pods to configure.</p>
</td>
</tr>
<tr>
<td>
<code>logs</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Log">
[]Log
</a>
</em>
</td>
<td>
<p>Forwards is a list of log configurations.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterLogsStatus">
ClusterLogsStatus
</a>
</em>
</td>
<td>
<p>Status holds status for cluster logs</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ClusterPortForward">
ClusterPortForward
<a href="#kwok.x-k8s.io%2fv1alpha1.ClusterPortForward"> #</a>
</h3>
<p>
<p>ClusterPortForward provides cluster-wide port forward configuration.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code>
string
</td>
<td>
<code>
kwok.x-k8s.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code>
string
</td>
<td><code>ClusterPortForward</code></td>
</tr>
<tr>
<td>
<code>metadata</code>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<p>Standard list metadata.
More info: <a href="https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata">https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata</a></p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterPortForwardSpec">
ClusterPortForwardSpec
</a>
</em>
</td>
<td>
<p>Spec holds spec for cluster port forward.</p>
<table>
<tr>
<td>
<code>selector</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ObjectSelector">
ObjectSelector
</a>
</em>
</td>
<td>
<p>Selector is a selector to filter pods to configure.</p>
</td>
</tr>
<tr>
<td>
<code>forwards</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Forward">
[]Forward
</a>
</em>
</td>
<td>
<p>Forwards is a list of forwards to configure.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterPortForwardStatus">
ClusterPortForwardStatus
</a>
</em>
</td>
<td>
<p>Status holds status for cluster port forward</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ClusterResourceUsage">
ClusterResourceUsage
<a href="#kwok.x-k8s.io%2fv1alpha1.ClusterResourceUsage"> #</a>
</h3>
<p>
<p>ClusterResourceUsage provides cluster-wide resource usage.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code>
string
</td>
<td>
<code>
kwok.x-k8s.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code>
string
</td>
<td><code>ClusterResourceUsage</code></td>
</tr>
<tr>
<td>
<code>metadata</code>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<p>Standard list metadata.
More info: <a href="https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata">https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata</a></p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterResourceUsageSpec">
ClusterResourceUsageSpec
</a>
</em>
</td>
<td>
<p>Spec holds spec for cluster resource usage.</p>
<table>
<tr>
<td>
<code>selector</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ObjectSelector">
ObjectSelector
</a>
</em>
</td>
<td>
<p>Selector is a selector to filter pods to configure.</p>
</td>
</tr>
<tr>
<td>
<code>usages</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ResourceUsageContainer">
[]ResourceUsageContainer
</a>
</em>
</td>
<td>
<p>Usages is a list of resource usage for the pod.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterResourceUsageStatus">
ClusterResourceUsageStatus
</a>
</em>
</td>
<td>
<p>Status holds status for cluster resource usage</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.Exec">
Exec
<a href="#kwok.x-k8s.io%2fv1alpha1.Exec"> #</a>
</h3>
<p>
<p>Exec provides exec configuration for a single pod.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code>
string
</td>
<td>
<code>
kwok.x-k8s.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code>
string
</td>
<td><code>Exec</code></td>
</tr>
<tr>
<td>
<code>metadata</code>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<p>Standard list metadata.
More info: <a href="https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata">https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata</a></p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ExecSpec">
ExecSpec
</a>
</em>
</td>
<td>
<p>Spec holds spec for exec</p>
<table>
<tr>
<td>
<code>execs</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ExecTarget">
[]ExecTarget
</a>
</em>
</td>
<td>
<p>Execs is a list of execs to configure.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ExecStatus">
ExecStatus
</a>
</em>
</td>
<td>
<p>Status holds status for exec</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.Logs">
Logs
<a href="#kwok.x-k8s.io%2fv1alpha1.Logs"> #</a>
</h3>
<p>
<p>Logs provides logging configuration for a single pod.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code>
string
</td>
<td>
<code>
kwok.x-k8s.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code>
string
</td>
<td><code>Logs</code></td>
</tr>
<tr>
<td>
<code>metadata</code>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<p>Standard list metadata.
More info: <a href="https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata">https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata</a></p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.LogsSpec">
LogsSpec
</a>
</em>
</td>
<td>
<p>Spec holds spec for logs</p>
<table>
<tr>
<td>
<code>logs</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Log">
[]Log
</a>
</em>
</td>
<td>
<p>Logs is a list of logs to configure.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.LogsStatus">
LogsStatus
</a>
</em>
</td>
<td>
<p>Status holds status for logs</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.Metric">
Metric
<a href="#kwok.x-k8s.io%2fv1alpha1.Metric"> #</a>
</h3>
<p>
<p>Metric provides metrics configuration.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code>
string
</td>
<td>
<code>
kwok.x-k8s.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code>
string
</td>
<td><code>Metric</code></td>
</tr>
<tr>
<td>
<code>metadata</code>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<p>Standard list metadata.
More info: <a href="https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata">https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata</a></p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.MetricSpec">
MetricSpec
</a>
</em>
</td>
<td>
<p>Spec holds spec for metrics.</p>
<table>
<tr>
<td>
<code>path</code>
<em>
string
</em>
</td>
<td>
<p>Path is a restful service path.</p>
</td>
</tr>
<tr>
<td>
<code>metrics</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.MetricConfig">
[]MetricConfig
</a>
</em>
</td>
<td>
<p>Metrics is a list of metric configurations.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.MetricStatus">
MetricStatus
</a>
</em>
</td>
<td>
<p>Status holds status for metrics</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.PortForward">
PortForward
<a href="#kwok.x-k8s.io%2fv1alpha1.PortForward"> #</a>
</h3>
<p>
<p>PortForward provides port forward configuration for a single pod.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code>
string
</td>
<td>
<code>
kwok.x-k8s.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code>
string
</td>
<td><code>PortForward</code></td>
</tr>
<tr>
<td>
<code>metadata</code>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<p>Standard list metadata.
More info: <a href="https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata">https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata</a></p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.PortForwardSpec">
PortForwardSpec
</a>
</em>
</td>
<td>
<p>Spec holds spec for port forward.</p>
<table>
<tr>
<td>
<code>forwards</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Forward">
[]Forward
</a>
</em>
</td>
<td>
<p>Forwards is a list of forwards to configure.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.PortForwardStatus">
PortForwardStatus
</a>
</em>
</td>
<td>
<p>Status holds status for port forward</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ResourceUsage">
ResourceUsage
<a href="#kwok.x-k8s.io%2fv1alpha1.ResourceUsage"> #</a>
</h3>
<p>
<p>ResourceUsage provides resource usage for a single pod.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code>
string
</td>
<td>
<code>
kwok.x-k8s.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code>
string
</td>
<td><code>ResourceUsage</code></td>
</tr>
<tr>
<td>
<code>metadata</code>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<p>Standard list metadata.
More info: <a href="https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata">https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata</a></p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ResourceUsageSpec">
ResourceUsageSpec
</a>
</em>
</td>
<td>
<p>Spec holds spec for resource usage.</p>
<table>
<tr>
<td>
<code>usages</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ResourceUsageContainer">
[]ResourceUsageContainer
</a>
</em>
</td>
<td>
<p>Usages is a list of resource usage for the pod.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ResourceUsageStatus">
ResourceUsageStatus
</a>
</em>
</td>
<td>
<p>Status holds status for resource usage</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.Stage">
Stage
<a href="#kwok.x-k8s.io%2fv1alpha1.Stage"> #</a>
</h3>
<p>
<p>Stage is an API that describes the staged change of a resource</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code>
string
</td>
<td>
<code>
kwok.x-k8s.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code>
string
</td>
<td><code>Stage</code></td>
</tr>
<tr>
<td>
<code>metadata</code>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<p>Standard list metadata.
More info: <a href="https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata">https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata</a></p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.StageSpec">
StageSpec
</a>
</em>
</td>
<td>
<p>Spec holds information about the request being evaluated.</p>
<table>
<tr>
<td>
<code>resourceRef</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.StageResourceRef">
StageResourceRef
</a>
</em>
</td>
<td>
<p>ResourceRef specifies the Kind and version of the resource.</p>
</td>
</tr>
<tr>
<td>
<code>selector</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.StageSelector">
StageSelector
</a>
</em>
</td>
<td>
<p>Selector specifies the stags will be applied to the selected resource.</p>
</td>
</tr>
<tr>
<td>
<code>weight</code>
<em>
int
</em>
</td>
<td>
<p>Weight means when multiple stages share the same ResourceRef and Selector,
a random stage will be matched as the next stage based on the weight.</p>
</td>
</tr>
<tr>
<td>
<code>weightFrom</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ExpressionFromSource">
ExpressionFromSource
</a>
</em>
</td>
<td>
<p>WeightFrom means is the expression used to get the value.
If it is a number type, convert to int.
If it is a string type, the value get will be parsed by strconv.ParseInt.</p>
</td>
</tr>
<tr>
<td>
<code>delay</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.StageDelay">
StageDelay
</a>
</em>
</td>
<td>
<p>Delay means there is a delay in this stage.</p>
</td>
</tr>
<tr>
<td>
<code>next</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.StageNext">
StageNext
</a>
</em>
</td>
<td>
<p>Next indicates that this stage will be moved to.</p>
</td>
</tr>
<tr>
<td>
<code>immediateNextStage</code>
<em>
bool
</em>
</td>
<td>
<p>ImmediateNextStage means that the next stage of matching is performed immediately, without waiting for the Apiserver to push.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.StageStatus">
StageStatus
</a>
</em>
</td>
<td>
<p>Status holds status for the Stage</p>
</td>
</tr>
</tbody>
</table>
<h2 id="references">
References
<a href="#references"> #</a>
</h2>
<h3 id="action.kwok.x-k8s.io/v1alpha1.GroupVersionResource">
GroupVersionResource
<a href="#action.kwok.x-k8s.io%2fv1alpha1.GroupVersionResource"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#action.kwok.x-k8s.io/v1alpha1.ResourcePatch">ResourcePatch</a>
</p>
<p>
<p>GroupVersionResource is a struct that represents the group version resource.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>group</code>
<em>
string
</em>
</td>
<td>
<p>Group represents the group of the resource.</p>
</td>
</tr>
<tr>
<td>
<code>version</code>
<em>
string
</em>
</td>
<td>
<p>Version represents the version of the resource.</p>
</td>
</tr>
<tr>
<td>
<code>resource</code>
<em>
string
</em>
</td>
<td>
<p>Resource represents the type of the resource.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="action.kwok.x-k8s.io/v1alpha1.PatchMethod">
PatchMethod
(<code>string</code> alias)
<a href="#action.kwok.x-k8s.io%2fv1alpha1.PatchMethod"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#action.kwok.x-k8s.io/v1alpha1.ResourcePatch">ResourcePatch</a>
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td><code>&#34;create&#34;</code></td>
<td><p>PatchMethodCreate means that the resource will be created by create.</p>
</td>
</tr>
<tr>
<td><code>&#34;delete&#34;</code></td>
<td><p>PatchMethodDelete means that the resource will be deleted by delete.</p>
</td>
</tr>
<tr>
<td><code>&#34;patch&#34;</code></td>
<td><p>PatchMethodPatch means that the resource will be patched by patch.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="action.kwok.x-k8s.io/v1alpha1.Target">
Target
<a href="#action.kwok.x-k8s.io%2fv1alpha1.Target"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#action.kwok.x-k8s.io/v1alpha1.ResourcePatch">ResourcePatch</a>
</p>
<p>
<p>Target is a struct that represents the target of the ResourcePatch.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code>
<em>
string
</em>
</td>
<td>
<p>Name represents the name of the resource to be patched.</p>
</td>
</tr>
<tr>
<td>
<code>namespace</code>
<em>
string
</em>
</td>
<td>
<p>Namespace represents the namespace of the resource to be patched.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.kwok.x-k8s.io/v1alpha1.Component">
Component
<a href="#config.kwok.x-k8s.io%2fv1alpha1.Component"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#config.kwok.x-k8s.io/v1alpha1.KwokctlConfiguration">KwokctlConfiguration</a>
</p>
<p>
<p>Component is a component of the cluster.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code>
<em>
string
</em>
</td>
<td>
<p>Name of the component specified as a DNS_LABEL.
Each component must have a unique name (DNS_LABEL).
Cannot be updated.</p>
</td>
</tr>
<tr>
<td>
<code>links</code>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Links is a set of links for the component.</p>
</td>
</tr>
<tr>
<td>
<code>binary</code>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Binary is the binary of the component.</p>
</td>
</tr>
<tr>
<td>
<code>image</code>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Image is the image of the component.</p>
</td>
</tr>
<tr>
<td>
<code>command</code>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Command is Entrypoint array. Not executed within a shell. Only works with Image.</p>
</td>
</tr>
<tr>
<td>
<code>user</code>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>User is the user for the component.</p>
</td>
</tr>
<tr>
<td>
<code>args</code>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Args is Arguments to the entrypoint.</p>
</td>
</tr>
<tr>
<td>
<code>workDir</code>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>WorkDir is component&rsquo;s working directory.</p>
</td>
</tr>
<tr>
<td>
<code>ports</code>
<em>
<a href="#config.kwok.x-k8s.io/v1alpha1.Port">
[]Port
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Ports is list of ports to expose from the component.</p>
</td>
</tr>
<tr>
<td>
<code>envs</code>
<em>
<a href="#config.kwok.x-k8s.io/v1alpha1.Env">
[]Env
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Envs is list of environment variables to set in the component.</p>
</td>
</tr>
<tr>
<td>
<code>volumes</code>
<em>
<a href="#config.kwok.x-k8s.io/v1alpha1.Volume">
[]Volume
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Volumes is a list of named volumes that can be mounted by containers belonging to the component.</p>
</td>
</tr>
<tr>
<td>
<code>metric</code>
<em>
<a href="#config.kwok.x-k8s.io/v1alpha1.ComponentMetric">
ComponentMetric
</a>
</em>
</td>
<td>
<p>Metric is the metric of the component.</p>
</td>
</tr>
<tr>
<td>
<code>metricsDiscovery</code>
<em>
<a href="#config.kwok.x-k8s.io/v1alpha1.ComponentMetric">
ComponentMetric
</a>
</em>
</td>
<td>
<p>MetricsDiscovery is the metrics discovery of the component.</p>
</td>
</tr>
<tr>
<td>
<code>address</code>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Address is the address of the component.</p>
</td>
</tr>
<tr>
<td>
<code>version</code>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Version is the version of the component.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.kwok.x-k8s.io/v1alpha1.ComponentMetric">
ComponentMetric
<a href="#config.kwok.x-k8s.io%2fv1alpha1.ComponentMetric"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#config.kwok.x-k8s.io/v1alpha1.Component">Component</a>
</p>
<p>
<p>ComponentMetric represents a metric of a component.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>scheme</code>
<em>
string
</em>
</td>
<td>
<p>Scheme is the scheme of the metric.</p>
</td>
</tr>
<tr>
<td>
<code>host</code>
<em>
string
</em>
</td>
<td>
<p>Host is the host of the metric.</p>
</td>
</tr>
<tr>
<td>
<code>path</code>
<em>
string
</em>
</td>
<td>
<p>Path is the path of the metric.</p>
</td>
</tr>
<tr>
<td>
<code>certPath</code>
<em>
string
</em>
</td>
<td>
<p>CertPath is the cert path of the metric.</p>
</td>
</tr>
<tr>
<td>
<code>keyPath</code>
<em>
string
</em>
</td>
<td>
<p>KeyPath is the key path of the metric.</p>
</td>
</tr>
<tr>
<td>
<code>insecureSkipVerify</code>
<em>
bool
</em>
</td>
<td>
<p>InsecureSkipVerify is the flag to skip verify the metric.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.kwok.x-k8s.io/v1alpha1.ComponentPatches">
ComponentPatches
<a href="#config.kwok.x-k8s.io%2fv1alpha1.ComponentPatches"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#config.kwok.x-k8s.io/v1alpha1.KwokctlConfiguration">KwokctlConfiguration</a>
</p>
<p>
<p>ComponentPatches holds information about the component patches.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the component.</p>
</td>
</tr>
<tr>
<td>
<code>extraArgs</code>
<em>
<a href="#config.kwok.x-k8s.io/v1alpha1.ExtraArgs">
[]ExtraArgs
</a>
</em>
</td>
<td>
<p>ExtraArgs is the extra args to be patched on the component.</p>
</td>
</tr>
<tr>
<td>
<code>extraVolumes</code>
<em>
<a href="#config.kwok.x-k8s.io/v1alpha1.Volume">
[]Volume
</a>
</em>
</td>
<td>
<p>ExtraVolumes is the extra volumes to be patched on the component.</p>
</td>
</tr>
<tr>
<td>
<code>extraEnvs</code>
<em>
<a href="#config.kwok.x-k8s.io/v1alpha1.Env">
[]Env
</a>
</em>
</td>
<td>
<p>ExtraEnvs is the extra environment variables to be patched on the component.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.kwok.x-k8s.io/v1alpha1.Env">
Env
<a href="#config.kwok.x-k8s.io%2fv1alpha1.Env"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#config.kwok.x-k8s.io/v1alpha1.Component">Component</a>
, 
<a href="#config.kwok.x-k8s.io/v1alpha1.ComponentPatches">ComponentPatches</a>
</p>
<p>
<p>Env represents an environment variable present in a Container.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code>
<em>
string
</em>
</td>
<td>
<p>Name of the environment variable.</p>
</td>
</tr>
<tr>
<td>
<code>value</code>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Value is using the previously defined environment variables in the component.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.kwok.x-k8s.io/v1alpha1.ExtraArgs">
ExtraArgs
<a href="#config.kwok.x-k8s.io%2fv1alpha1.ExtraArgs"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#config.kwok.x-k8s.io/v1alpha1.ComponentPatches">ComponentPatches</a>
</p>
<p>
<p>ExtraArgs holds information about the extra args.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>key</code>
<em>
string
</em>
</td>
<td>
<p>Key is the key of the extra args.</p>
</td>
</tr>
<tr>
<td>
<code>value</code>
<em>
string
</em>
</td>
<td>
<p>Value is the value of the extra args.</p>
</td>
</tr>
<tr>
<td>
<code>override</code>
<em>
bool
</em>
</td>
<td>
<p>Override is the value of is it override the arg</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.kwok.x-k8s.io/v1alpha1.HostPathType">
HostPathType
(<code>string</code> alias)
<a href="#config.kwok.x-k8s.io%2fv1alpha1.HostPathType"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#config.kwok.x-k8s.io/v1alpha1.Volume">Volume</a>
</p>
<p>
<p>HostPathType represents the type of storage used for HostPath volumes.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td><code>&#34;BlockDevice&#34;</code></td>
<td><p>A block device must exist at the given path</p>
</td>
</tr>
<tr>
<td><code>&#34;CharDevice&#34;</code></td>
<td><p>A character device must exist at the given path</p>
</td>
</tr>
<tr>
<td><code>&#34;Directory&#34;</code></td>
<td><p>A directory must exist at the given path</p>
</td>
</tr>
<tr>
<td><code>&#34;DirectoryOrCreate&#34;</code></td>
<td><p>If nothing exists at the given path, an empty directory will be created there
as needed with file mode 0755, having the same group and ownership with Kubelet.</p>
</td>
</tr>
<tr>
<td><code>&#34;File&#34;</code></td>
<td><p>A file must exist at the given path</p>
</td>
</tr>
<tr>
<td><code>&#34;FileOrCreate&#34;</code></td>
<td><p>If nothing exists at the given path, an empty file will be created there
as needed with file mode 0644, having the same group and ownership with Kubelet.</p>
</td>
</tr>
<tr>
<td><code>&#34;Socket&#34;</code></td>
<td><p>A UNIX socket must exist at the given path</p>
</td>
</tr>
<tr>
<td><code>&#34;&#34;</code></td>
<td><p>For backwards compatible, leave it empty if unset</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.kwok.x-k8s.io/v1alpha1.KwokConfigurationOptions">
KwokConfigurationOptions
<a href="#config.kwok.x-k8s.io%2fv1alpha1.KwokConfigurationOptions"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#config.kwok.x-k8s.io/v1alpha1.KwokConfiguration">KwokConfiguration</a>
</p>
<p>
<p>KwokConfigurationOptions holds information about the options.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>enableCRDs</code>
<em>
[]string
</em>
</td>
<td>
<p>EnableCRDs is a list of CRDs to enable.
Once listed in this field, it will no longer be supported by the &ndash;config flag.</p>
</td>
</tr>
<tr>
<td>
<code>cidr</code>
<em>
string
</em>
</td>
<td>
<p>The default IP assigned to the Pod on maintained Nodes.
is the default value for flag &ndash;cidr</p>
</td>
</tr>
<tr>
<td>
<code>nodeIP</code>
<em>
string
</em>
</td>
<td>
<p>The ip of all nodes maintained by the Kwok
is the default value for flag &ndash;node-ip</p>
</td>
</tr>
<tr>
<td>
<code>nodeName</code>
<em>
string
</em>
</td>
<td>
<p>The name of all nodes maintained by the Kwok
is the default value for flag &ndash;node-name</p>
</td>
</tr>
<tr>
<td>
<code>nodePort</code>
<em>
int
</em>
</td>
<td>
<p>The port of all nodes maintained by the Kwok
is the default value for flag &ndash;node-port</p>
</td>
</tr>
<tr>
<td>
<code>tlsCertFile</code>
<em>
string
</em>
</td>
<td>
<p>TLSCertFile is the file containing x509 Certificate for HTTPS.
If HTTPS serving is enabled, and &ndash;tls-cert-file and &ndash;tls-private-key-file
is the default value for flag &ndash;tls-cert-file</p>
</td>
</tr>
<tr>
<td>
<code>tlsPrivateKeyFile</code>
<em>
string
</em>
</td>
<td>
<p>TLSPrivateKeyFile is the ile containing x509 private key matching &ndash;tls-cert-file.
is the default value for flag &ndash;tls-private-key-file</p>
</td>
</tr>
<tr>
<td>
<code>manageSingleNode</code>
<em>
string
</em>
</td>
<td>
<p>ManageSingleNode is the option to manage a single node name.
is the default value for flag &ndash;manage-single-node
Note: when <code>manage-all-nodes</code> is specified as true or
<code>manage-nodes-with-label-selector</code> or <code>manage-nodes-with-annotation-selector</code> is specified,
this is a no-op.</p>
</td>
</tr>
<tr>
<td>
<code>manageAllNodes</code>
<em>
bool
</em>
</td>
<td>
<p>Default option to manage (i.e., maintain heartbeat/liveness of) all Nodes or not.
is the default value for flag &ndash;manage-all-nodes
Note: when <code>manage-single-node</code> is specified as true or
<code>manage-nodes-with-label-selector</code> or <code>manage-nodes-with-annotation-selector</code> is specified,
this is a no-op.</p>
</td>
</tr>
<tr>
<td>
<code>manageNodesWithAnnotationSelector</code>
<em>
string
</em>
</td>
<td>
<p>Default annotations specified on Nodes to demand manage.
is the default value for flag &ndash;manage-nodes-with-annotation-selector
Note: when <code>all-node-manage</code> is specified as true or
<code>manage-single-node</code> is specified,
this is a no-op.</p>
</td>
</tr>
<tr>
<td>
<code>manageNodesWithLabelSelector</code>
<em>
string
</em>
</td>
<td>
<p>Default labels specified on Nodes to demand manage.
is the default value for flag &ndash;manage-nodes-with-label-selector
Note: when <code>all-node-manage</code> is specified as true or
<code>manage-single-node</code> is specified,
this is a no-op.</p>
</td>
</tr>
<tr>
<td>
<code>disregardStatusWithAnnotationSelector</code>
<em>
string
</em>
</td>
<td>
<p>If a Node/Pod is on a managed Node and has this annotation status will not be modified
is the default value for flag &ndash;disregard-status-with-annotation-selector
Deprecated: use Stage API instead</p>
</td>
</tr>
<tr>
<td>
<code>disregardStatusWithLabelSelector</code>
<em>
string
</em>
</td>
<td>
<p>If a Node/Pod is on a managed Node and has this label status will not be modified
is the default value for flag &ndash;disregard-status-with-label-selector
Deprecated: use Stage API instead</p>
</td>
</tr>
<tr>
<td>
<code>serverAddress</code>
<em>
string
</em>
</td>
<td>
<p>ServerAddress is server address of the Kwok.
is the default value for flag &ndash;server-address</p>
</td>
</tr>
<tr>
<td>
<code>experimentalEnableCNI</code>
<em>
bool
</em>
</td>
<td>
<p>Experimental support for getting pod ip from CNI, for CNI-related components, Only works with Linux.
is the default value for flag &ndash;experimental-enable-cni
Deprecated: It will be removed and will be supported in the form of plugins</p>
</td>
</tr>
<tr>
<td>
<code>enableDebuggingHandlers</code>
<em>
bool
</em>
</td>
<td>
<p>enableDebuggingHandlers enables server endpoints for log collection
and local running of containers and commands</p>
</td>
</tr>
<tr>
<td>
<code>enableContentionProfiling</code>
<em>
bool
</em>
</td>
<td>
<p>enableContentionProfiling enables lock contention profiling, if enableDebuggingHandlers is true.</p>
</td>
</tr>
<tr>
<td>
<code>enableProfilingHandler</code>
<em>
bool
</em>
</td>
<td>
<p>EnableProfiling enables /debug/pprof handler, if enableDebuggingHandlers is true.</p>
</td>
</tr>
<tr>
<td>
<code>podPlayStageParallelism</code>
<em>
uint
</em>
</td>
<td>
<p>PodPlayStageParallelism is the number of PodPlayStages that are allowed to run in parallel.</p>
</td>
</tr>
<tr>
<td>
<code>nodePlayStageParallelism</code>
<em>
uint
</em>
</td>
<td>
<p>NodePlayStageParallelism is the number of NodePlayStages that are allowed to run in parallel.</p>
</td>
</tr>
<tr>
<td>
<code>nodeLeaseDurationSeconds</code>
<em>
uint
</em>
</td>
<td>
<p>NodeLeaseDurationSeconds is the duration the Kubelet will set on its corresponding Lease.</p>
</td>
</tr>
<tr>
<td>
<code>nodeLeaseParallelism</code>
<em>
uint
</em>
</td>
<td>
<p>NodeLeaseParallelism is the number of NodeLeases that are allowed to be processed in parallel.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.kwok.x-k8s.io/v1alpha1.KwokctlConfigurationOptions">
KwokctlConfigurationOptions
<a href="#config.kwok.x-k8s.io%2fv1alpha1.KwokctlConfigurationOptions"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#config.kwok.x-k8s.io/v1alpha1.KwokctlConfiguration">KwokctlConfiguration</a>
</p>
<p>
<p>KwokctlConfigurationOptions holds information about the options.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>enableCRDs</code>
<em>
[]string
</em>
</td>
<td>
<p>EnableCRDs is a list of CRDs to enable.
Once listed in this field, it will no longer be supported by the &ndash;config flag.</p>
</td>
</tr>
<tr>
<td>
<code>kubeApiserverPort</code>
<em>
uint32
</em>
</td>
<td>
<p>KubeApiserverPort is the port to expose apiserver.
is the default value for flag &ndash;kube-apiserver-port and env KWOK_KUBE_APISERVER_PORT</p>
</td>
</tr>
<tr>
<td>
<code>kubeApiserverInsecurePort</code>
<em>
uint32
</em>
</td>
<td>
<p>KubeApiserverInsecurePort is the port to expose insecure apiserver.
is the default value for flag &ndash;kube-apiserver-insecure-port and env KWOK_KUBE_APISERVER_INSECURE_PORT</p>
</td>
</tr>
<tr>
<td>
<code>insecureKubeconfig</code>
<em>
bool
</em>
</td>
<td>
<p>InsecureKubeconfig is the flag to use insecure kubeconfig.
only available when KubeApiserverInsecurePort is set.</p>
</td>
</tr>
<tr>
<td>
<code>runtime</code>
<em>
string
</em>
</td>
<td>
<p>Runtime is the runtime to use.
is the default value for flag &ndash;runtime and env KWOK_RUNTIME</p>
</td>
</tr>
<tr>
<td>
<code>runtimes</code>
<em>
[]string
</em>
</td>
<td>
<p>Runtimes is a list of alternate runtimes. When Runtime is empty,
the availability of the runtimes in the list is checked one by one
and set to Runtime</p>
</td>
</tr>
<tr>
<td>
<code>prometheusPort</code>
<em>
uint32
</em>
</td>
<td>
<p>PrometheusPort is the port to expose Prometheus metrics.
is the default value for flag &ndash;prometheus-port and env KWOK_PROMETHEUS_PORT</p>
</td>
</tr>
<tr>
<td>
<code>jaegerPort</code>
<em>
uint32
</em>
</td>
<td>
<p>JaegerPort is the port to expose Jaeger UI.
is the default value for flag &ndash;jaeger-port and env KWOK_JAEGER_PORT</p>
</td>
</tr>
<tr>
<td>
<code>jaegerOtlpGrpcPort</code>
<em>
uint32
</em>
</td>
<td>
<p>JaegerOtlpGrpcPort is the port to expose OTLP GRPC collector.</p>
</td>
</tr>
<tr>
<td>
<code>kwokVersion</code>
<em>
string
</em>
</td>
<td>
<p>KwokVersion is the version of Kwok to use.
is the default value for env KWOK_VERSION</p>
</td>
</tr>
<tr>
<td>
<code>kubeVersion</code>
<em>
string
</em>
</td>
<td>
<p>KubeVersion is the version of Kubernetes to use.
is the default value for env KWOK_KUBE_VERSION</p>
</td>
</tr>
<tr>
<td>
<code>etcdVersion</code>
<em>
string
</em>
</td>
<td>
<p>EtcdVersion is the version of Etcd to use.
is the default value for env KWOK_ETCD_VERSION</p>
</td>
</tr>
<tr>
<td>
<code>dashboardVersion</code>
<em>
string
</em>
</td>
<td>
<p>DashboardVersion is the version of Kubernetes dashboard to use.</p>
</td>
</tr>
<tr>
<td>
<code>dashboardMetricsScraperVersion</code>
<em>
string
</em>
</td>
<td>
<p>DashboardMetricsScraperVersion is the version of Kubernetes dashboard metrics scraper to use.</p>
</td>
</tr>
<tr>
<td>
<code>prometheusVersion</code>
<em>
string
</em>
</td>
<td>
<p>PrometheusVersion is the version of Prometheus to use.
is the default value for env KWOK_PROMETHEUS_VERSION</p>
</td>
</tr>
<tr>
<td>
<code>jaegerVersion</code>
<em>
string
</em>
</td>
<td>
<p>JaegerVersion is the version of Jaeger to use.
is the default value for env KWOK_JAEGER_VERSION</p>
</td>
</tr>
<tr>
<td>
<code>metricsServerVersion</code>
<em>
string
</em>
</td>
<td>
<p>MetricsServerVersion is the version of metrics-server to use.</p>
</td>
</tr>
<tr>
<td>
<code>kindVersion</code>
<em>
string
</em>
</td>
<td>
<p>KindVersion is the version of kind to use.
is the default value for env KWOK_KIND_VERSION</p>
</td>
</tr>
<tr>
<td>
<code>securePort</code>
<em>
bool
</em>
</td>
<td>
<p>SecurePort is the apiserver port on which to serve HTTPS with authentication and authorization.
is not available before Kubernetes 1.13.0
is the default value for flag &ndash;secure-port and env KWOK_SECURE_PORT</p>
</td>
</tr>
<tr>
<td>
<code>quietPull</code>
<em>
bool
</em>
</td>
<td>
<p>QuietPull is the flag to quiet the pull.
is the default value for flag &ndash;quiet-pull and env KWOK_QUIET_PULL</p>
</td>
</tr>
<tr>
<td>
<code>kubeSchedulerConfig</code>
<em>
string
</em>
</td>
<td>
<p>KubeSchedulerConfig is the configuration path for kube-scheduler.
is the default value for flag &ndash;kube-scheduler-config and env KWOK_KUBE_SCHEDULER_CONFIG</p>
</td>
</tr>
<tr>
<td>
<code>components</code>
<em>
[]string
</em>
</td>
<td>
<p>Components is the configuration for components.</p>
</td>
</tr>
<tr>
<td>
<code>disable</code>
<em>
[]string
</em>
</td>
<td>
<p>Disable is the configuration for disables components.</p>
</td>
</tr>
<tr>
<td>
<code>enable</code>
<em>
[]string
</em>
</td>
<td>
<p>Enable is the configuration for enables components.</p>
</td>
</tr>
<tr>
<td>
<code>disableKubeScheduler</code>
<em>
bool
</em>
</td>
<td>
<p>DisableKubeScheduler is the flag to disable kube-scheduler.
is the default value for flag &ndash;disable-kube-scheduler and env KWOK_DISABLE_KUBE_SCHEDULER
Deprecated: Use Disable instead</p>
</td>
</tr>
<tr>
<td>
<code>disableKubeControllerManager</code>
<em>
bool
</em>
</td>
<td>
<p>DisableKubeControllerManager is the flag to disable kube-controller-manager.
is the default value for flag &ndash;disable-kube-controller-manager and env KWOK_DISABLE_KUBE_CONTROLLER_MANAGER
Deprecated: Use Disable instead</p>
</td>
</tr>
<tr>
<td>
<code>enableMetricsServer</code>
<em>
bool
</em>
</td>
<td>
<p>EnableMetricsServer is the flag to enable metrics-server.
Deprecated: Use Enable instead</p>
</td>
</tr>
<tr>
<td>
<code>kubeImagePrefix</code>
<em>
string
</em>
</td>
<td>
<p>KubeImagePrefix is the prefix of the kubernetes image.
is the default value for env KWOK_KUBE_IMAGE_PREFIX</p>
</td>
</tr>
<tr>
<td>
<code>etcdImagePrefix</code>
<em>
string
</em>
</td>
<td>
<p>EtcdImagePrefix is the prefix of the etcd image.
is the default value for env KWOK_ETCD_IMAGE_PREFIX</p>
</td>
</tr>
<tr>
<td>
<code>kwokImagePrefix</code>
<em>
string
</em>
</td>
<td>
<p>KwokImagePrefix is the prefix of the kwok image.
is the default value for env KWOK_IMAGE_PREFIX</p>
</td>
</tr>
<tr>
<td>
<code>dashboardImagePrefix</code>
<em>
string
</em>
</td>
<td>
<p>DashboardImagePrefix is the prefix of the dashboard image.</p>
</td>
</tr>
<tr>
<td>
<code>prometheusImagePrefix</code>
<em>
string
</em>
</td>
<td>
<p>PrometheusImagePrefix is the prefix of the Prometheus image.
is the default value for env KWOK_PROMETHEUS_IMAGE_PREFIX</p>
</td>
</tr>
<tr>
<td>
<code>jaegerImagePrefix</code>
<em>
string
</em>
</td>
<td>
<p>JaegerImagePrefix is the prefix of the Jaeger image.
is the default value for env KWOK_JAEGER_IMAGE_PREFIX</p>
</td>
</tr>
<tr>
<td>
<code>metricsServerImagePrefix</code>
<em>
string
</em>
</td>
<td>
<p>MetricsServerImagePrefix is the prefix of the metrics-server image.</p>
</td>
</tr>
<tr>
<td>
<code>etcdImage</code>
<em>
string
</em>
</td>
<td>
<p>EtcdImage is the image of etcd.
is the default value for flag &ndash;etcd-image and env KWOK_ETCD_IMAGE</p>
</td>
</tr>
<tr>
<td>
<code>kubeApiserverImage</code>
<em>
string
</em>
</td>
<td>
<p>KubeApiserverImage is the image of kube-apiserver.
is the default value for flag &ndash;kube-apiserver-image and env KWOK_KUBE_APISERVER_IMAGE</p>
</td>
</tr>
<tr>
<td>
<code>kubeControllerManagerImage</code>
<em>
string
</em>
</td>
<td>
<p>KubeControllerManagerImage is the image of kube-controller-manager.
is the default value for flag &ndash;kube-controller-manager-image and env KWOK_KUBE_CONTROLLER_MANAGER_IMAGE</p>
</td>
</tr>
<tr>
<td>
<code>kubeSchedulerImage</code>
<em>
string
</em>
</td>
<td>
<p>KubeSchedulerImage is the image of kube-scheduler.
is the default value for flag &ndash;kube-scheduler-image and env KWOK_KUBE_SCHEDULER_IMAGE</p>
</td>
</tr>
<tr>
<td>
<code>kubectlImage</code>
<em>
string
</em>
</td>
<td>
<p>KubectlImage is the image of kubectl.
is the default value for flag &ndash;kubectl-image and env KWOK_KUBECTL_IMAGE</p>
</td>
</tr>
<tr>
<td>
<code>kwokControllerImage</code>
<em>
string
</em>
</td>
<td>
<p>KwokControllerImage is the image of Kwok.
is the default value for flag &ndash;controller-image and env KWOK_CONTROLLER_IMAGE</p>
</td>
</tr>
<tr>
<td>
<code>dashboardImage</code>
<em>
string
</em>
</td>
<td>
<p>DashboardImage is the image of dashboard.</p>
</td>
</tr>
<tr>
<td>
<code>dashboardMetricsScraperImage</code>
<em>
string
</em>
</td>
<td>
<p>DashboardMetricsScraperImage is the image of dashboard metrics scraper.</p>
</td>
</tr>
<tr>
<td>
<code>prometheusImage</code>
<em>
string
</em>
</td>
<td>
<p>PrometheusImage is the image of Prometheus.
is the default value for flag &ndash;prometheus-image and env KWOK_PROMETHEUS_IMAGE</p>
</td>
</tr>
<tr>
<td>
<code>jaegerImage</code>
<em>
string
</em>
</td>
<td>
<p>JaegerImage is the image of Jaeger.
is the default value for flag &ndash;jaeger-image and env KWOK_JAEGER_IMAGE</p>
</td>
</tr>
<tr>
<td>
<code>metricsServerImage</code>
<em>
string
</em>
</td>
<td>
<p>MetricsServerImage is the image of metrics-server.</p>
</td>
</tr>
<tr>
<td>
<code>kindNodeImagePrefix</code>
<em>
string
</em>
</td>
<td>
<p>KindNodeImagePrefix is the prefix of the kind node image.
is the default value for env KWOK_KIND_NODE_IMAGE_PREFIX</p>
</td>
</tr>
<tr>
<td>
<code>kindNodeImage</code>
<em>
string
</em>
</td>
<td>
<p>KindNodeImage is the image of kind node.
is the default value for flag &ndash;kind-node-image and env KWOK_KIND_NODE_IMAGE</p>
</td>
</tr>
<tr>
<td>
<code>binSuffix</code>
<em>
string
</em>
</td>
<td>
<p>BinSuffix is the suffix of the all binary.
On Windows is .exe</p>
</td>
</tr>
<tr>
<td>
<code>kubeBinaryPrefix</code>
<em>
string
</em>
</td>
<td>
<p>KubeBinaryPrefix is the prefix of the kubernetes binary.
is the default value for env KWOK_KUBE_BINARY_PREFIX</p>
</td>
</tr>
<tr>
<td>
<code>kubeApiserverBinary</code>
<em>
string
</em>
</td>
<td>
<p>KubeApiserverBinary is the binary of kube-apiserver.
is the default value for flag &ndash;apiserver-binary and env KWOK_KUBE_APISERVER_BINARY</p>
</td>
</tr>
<tr>
<td>
<code>kubeControllerManagerBinary</code>
<em>
string
</em>
</td>
<td>
<p>KubeControllerManagerBinary is the binary of kube-controller-manager.
is the default value for flag &ndash;controller-manager-binary and env KWOK_KUBE_CONTROLLER_MANAGER_BINARY</p>
</td>
</tr>
<tr>
<td>
<code>kubeSchedulerBinary</code>
<em>
string
</em>
</td>
<td>
<p>KubeSchedulerBinary is the binary of kube-scheduler.
is the default value for flag &ndash;scheduler-binary and env KWOK_KUBE_SCHEDULER_BINARY</p>
</td>
</tr>
<tr>
<td>
<code>kubectlBinary</code>
<em>
string
</em>
</td>
<td>
<p>KubectlBinary is the binary of kubectl.
is the default value for env KWOK_KUBECTL_BINARY</p>
</td>
</tr>
<tr>
<td>
<code>etcdBinaryPrefix</code>
<em>
string
</em>
</td>
<td>
<p>EtcdBinaryPrefix is the prefix of the etcd binary.
is the default value for env KWOK_ETCD_BINARY_PREFIX</p>
</td>
</tr>
<tr>
<td>
<code>etcdctlBinary</code>
<em>
string
</em>
</td>
<td>
<p>EtcdctlBinary is the binary of etcdctl.</p>
</td>
</tr>
<tr>
<td>
<code>etcdBinary</code>
<em>
string
</em>
</td>
<td>
<p>EtcdBinary is the binary of etcd.
is the default value for flag &ndash;etcd-binary and env KWOK_ETCD_BINARY</p>
</td>
</tr>
<tr>
<td>
<code>etcdBinaryTar</code>
<em>
string
</em>
</td>
<td>
<p>EtcdBinaryTar is the tar of the binary of etcd.
is the default value for env KWOK_ETCD_BINARY_TAR
Deprecated: Use EtcdBinary or EtcdctlBinary instead</p>
</td>
</tr>
<tr>
<td>
<code>etcdPrefix</code>
<em>
string
</em>
</td>
<td>
<p>EtcdPrefix is the prefix of etcd.</p>
</td>
</tr>
<tr>
<td>
<code>kwokBinaryPrefix</code>
<em>
string
</em>
</td>
<td>
<p>KwokBinaryPrefix is the prefix of the kwok binary.
is the default value for env KWOK_BINARY_PREFIX</p>
</td>
</tr>
<tr>
<td>
<code>kwokControllerBinary</code>
<em>
string
</em>
</td>
<td>
<p>KwokControllerBinary is the binary of kwok.
is the default value for flag &ndash;controller-binary and env KWOK_CONTROLLER_BINARY</p>
</td>
</tr>
<tr>
<td>
<code>prometheusBinaryPrefix</code>
<em>
string
</em>
</td>
<td>
<p>PrometheusBinaryPrefix is the prefix of the Prometheus binary.
is the default value for env KWOK_PROMETHEUS_PREFIX</p>
</td>
</tr>
<tr>
<td>
<code>prometheusBinary</code>
<em>
string
</em>
</td>
<td>
<p>PrometheusBinary  is the binary of Prometheus.
is the default value for flag &ndash;prometheus-binary and env KWOK_PROMETHEUS_BINARY</p>
</td>
</tr>
<tr>
<td>
<code>prometheusBinaryTar</code>
<em>
string
</em>
</td>
<td>
<p>PrometheusBinaryTar is the tar of binary of Prometheus.
is the default value for env KWOK_PROMETHEUS_BINARY_TAR
Deprecated: Use PrometheusBinary instead</p>
</td>
</tr>
<tr>
<td>
<code>jaegerBinaryPrefix</code>
<em>
string
</em>
</td>
<td>
<p>JaegerBinaryPrefix is the prefix of the Jaeger binary.
is the default value for env KWOK_JAEGER_PREFIX</p>
</td>
</tr>
<tr>
<td>
<code>jaegerBinary</code>
<em>
string
</em>
</td>
<td>
<p>JaegerBinary  is the binary of Jaeger.
is the default value for flag &ndash;jaeger-binary and env KWOK_JAEGER_BINARY</p>
</td>
</tr>
<tr>
<td>
<code>jaegerBinaryTar</code>
<em>
string
</em>
</td>
<td>
<p>JaegerBinaryTar is the tar of binary of Jaeger.
is the default value for env KWOK_JAEGER_TAR
Deprecated: Use JaegerBinary instead</p>
</td>
</tr>
<tr>
<td>
<code>metricsServerBinaryPrefix</code>
<em>
string
</em>
</td>
<td>
<p>MetricsServerBinaryPrefix is the prefix of the metrics-server binary.</p>
</td>
</tr>
<tr>
<td>
<code>metricsServerBinary</code>
<em>
string
</em>
</td>
<td>
<p>MetricsServerBinary is the binary of metrics-server.</p>
</td>
</tr>
<tr>
<td>
<code>kindBinaryPrefix</code>
<em>
string
</em>
</td>
<td>
<p>KindBinaryPrefix is the binary prefix of kind.
is the default value for env KWOK_KIND_BINARY_PREFIX</p>
</td>
</tr>
<tr>
<td>
<code>kindBinary</code>
<em>
string
</em>
</td>
<td>
<p>KindBinary is the binary of kind.
is the default value for flag &ndash;kind-binary and env KWOK_KIND_BINARY</p>
</td>
</tr>
<tr>
<td>
<code>mode</code>
<em>
string
</em>
</td>
<td>
<p>Mode is several default parameter templates for clusters
is the default value for env KWOK_MODE
k8s 1.29, different components use different FeatureGate,
which makes it impossible to create clusters properly using this feature.
Deprecated: This mode will be removed in a future release</p>
</td>
</tr>
<tr>
<td>
<code>kubeFeatureGates</code>
<em>
string
</em>
</td>
<td>
<p>KubeFeatureGates is a set of key=value pairs that describe feature gates for alpha/experimental features of Kubernetes.
is the default value for flag &ndash;kube-feature-gates and env KWOK_KUBE_FEATURE_DATES</p>
</td>
</tr>
<tr>
<td>
<code>kubeRuntimeConfig</code>
<em>
string
</em>
</td>
<td>
<p>KubeRuntimeConfig is a set of key=value pairs that enable or disable built-in APIs.
is the default value for flag &ndash;kube-runtime-config and env KWOK_KUBE_RUNTIME_CONFIG</p>
</td>
</tr>
<tr>
<td>
<code>kubeAuditPolicy</code>
<em>
string
</em>
</td>
<td>
<p>KubeAuditPolicy is path to the file that defines the audit policy configuration
is the default value for flag &ndash;kube-audit-policy and env KWOK_KUBE_AUDIT_POLICY</p>
</td>
</tr>
<tr>
<td>
<code>kubeAuthorization</code>
<em>
bool
</em>
</td>
<td>
<p>KubeAuthorization is the flag to enable authorization on secure port.
is the default value for flag &ndash;kube-authorization and env KWOK_KUBE_AUTHORIZATION</p>
</td>
</tr>
<tr>
<td>
<code>kubeAdmission</code>
<em>
bool
</em>
</td>
<td>
<p>KubeAdmission is the flag to enable admission for kube-apiserver.
is the default value for flag &ndash;kube-admission and env KWOK_KUBE_ADMISSION</p>
</td>
</tr>
<tr>
<td>
<code>etcdPeerPort</code>
<em>
uint32
</em>
</td>
<td>
<p>EtcdPeerPort is etcd peer port in the binary runtime</p>
</td>
</tr>
<tr>
<td>
<code>etcdPort</code>
<em>
uint32
</em>
</td>
<td>
<p>EtcdPort is etcd port in the binary runtime</p>
</td>
</tr>
<tr>
<td>
<code>kubeControllerManagerPort</code>
<em>
uint32
</em>
</td>
<td>
<p>KubeControllerManagerPort is kube-controller-manager port in the binary runtime</p>
</td>
</tr>
<tr>
<td>
<code>kubeSchedulerPort</code>
<em>
uint32
</em>
</td>
<td>
<p>KubeSchedulerPort is kube-scheduler port in the binary runtime</p>
</td>
</tr>
<tr>
<td>
<code>dashboardPort</code>
<em>
uint32
</em>
</td>
<td>
<p>DashboardPort is dashboard port in the binary runtime</p>
</td>
</tr>
<tr>
<td>
<code>kwokControllerPort</code>
<em>
uint32
</em>
</td>
<td>
<p>KwokControllerPort is kwok-controller port that is exposed to the host.
is the default value for flag &ndash;controller-port and env KWOK_CONTROLLER_PORT</p>
</td>
</tr>
<tr>
<td>
<code>metricsServerPort</code>
<em>
uint32
</em>
</td>
<td>
<p>MetricsServerPort is metrics-server port that is exposed to the host.</p>
</td>
</tr>
<tr>
<td>
<code>cacheDir</code>
<em>
string
</em>
</td>
<td>
<p>CacheDir is the directory of the cache.</p>
</td>
</tr>
<tr>
<td>
<code>kubeControllerManagerNodeMonitorPeriodMilliseconds</code>
<em>
int64
</em>
</td>
<td>
<p>KubeControllerManagerNodeMonitorPeriodMilliseconds is &ndash;node-monitor-period for kube-controller-manager.</p>
</td>
</tr>
<tr>
<td>
<code>kubeControllerManagerNodeMonitorGracePeriodMilliseconds</code>
<em>
int64
</em>
</td>
<td>
<p>KubeControllerManagerNodeMonitorGracePeriodMilliseconds is &ndash;node-monitor-grace-period for kube-controller-manager.</p>
</td>
</tr>
<tr>
<td>
<code>nodeStatusUpdateFrequencyMilliseconds</code>
<em>
int64
</em>
</td>
<td>
<p>NodeStatusUpdateFrequencyMilliseconds is &ndash;node-status-update-frequency for kwok like kubelet.</p>
</td>
</tr>
<tr>
<td>
<code>nodeLeaseDurationSeconds</code>
<em>
uint
</em>
</td>
<td>
<p>NodeLeaseDurationSeconds is the duration the Kubelet will set on its corresponding Lease.</p>
</td>
</tr>
<tr>
<td>
<code>heartbeatFactor</code>
<em>
float64
</em>
</td>
<td>
<p>HeartbeatFactor is the scale factor for all about heartbeat.</p>
</td>
</tr>
<tr>
<td>
<code>bindAddress</code>
<em>
string
</em>
</td>
<td>
<p>BindAddress is the address to bind to.</p>
</td>
</tr>
<tr>
<td>
<code>kubeApiserverCertSANs</code>
<em>
[]string
</em>
</td>
<td>
<p>KubeApiserverCertSANs sets extra Subject Alternative Names for the API Server signing cert.</p>
</td>
</tr>
<tr>
<td>
<code>disableQPSLimits</code>
<em>
bool
</em>
</td>
<td>
<p>DisableQPSLimits specifies whether to disable QPS limits for components.</p>
</td>
</tr>
<tr>
<td>
<code>etcdQuotaBackendSize</code>
<em>
string
</em>
</td>
<td>
<p>EtcdQuotaBackendSize is the backend quota for etcd.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.kwok.x-k8s.io/v1alpha1.KwokctlConfigurationStatus">
KwokctlConfigurationStatus
<a href="#config.kwok.x-k8s.io%2fv1alpha1.KwokctlConfigurationStatus"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#config.kwok.x-k8s.io/v1alpha1.KwokctlConfiguration">KwokctlConfiguration</a>
</p>
<p>
<p>KwokctlConfigurationStatus holds information about the status.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>version</code>
<em>
string
</em>
</td>
<td>
<p>Version is the version of the kwokctl.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.kwok.x-k8s.io/v1alpha1.Port">
Port
<a href="#config.kwok.x-k8s.io%2fv1alpha1.Port"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#config.kwok.x-k8s.io/v1alpha1.Component">Component</a>
</p>
<p>
<p>Port represents a network port in a single component.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Name for the port that can be referred to by components.</p>
</td>
</tr>
<tr>
<td>
<code>port</code>
<em>
uint32
</em>
</td>
<td>
<p>Port is number of port to expose on the component&rsquo;s IP address.
This must be a valid port number, 0 &lt; x &lt; 65536.</p>
</td>
</tr>
<tr>
<td>
<code>hostPort</code>
<em>
uint32
</em>
</td>
<td>
<em>(Optional)</em>
<p>HostPort is number of port to expose on the host.
If specified, this must be a valid port number, 0 &lt; x &lt; 65536.</p>
</td>
</tr>
<tr>
<td>
<code>protocol</code>
<em>
<a href="#config.kwok.x-k8s.io/v1alpha1.Protocol">
Protocol
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Protocol for port. Must be UDP, TCP, or SCTP.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.kwok.x-k8s.io/v1alpha1.Protocol">
Protocol
(<code>string</code> alias)
<a href="#config.kwok.x-k8s.io%2fv1alpha1.Protocol"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#config.kwok.x-k8s.io/v1alpha1.Port">Port</a>
</p>
<p>
<p>Protocol defines network protocols supported for things like component ports.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td><code>&#34;SCTP&#34;</code></td>
<td><p>ProtocolSCTP is the SCTP protocol.</p>
</td>
</tr>
<tr>
<td><code>&#34;TCP&#34;</code></td>
<td><p>ProtocolTCP is the TCP protocol.</p>
</td>
</tr>
<tr>
<td><code>&#34;UDP&#34;</code></td>
<td><p>ProtocolUDP is the UDP protocol.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.kwok.x-k8s.io/v1alpha1.Volume">
Volume
<a href="#config.kwok.x-k8s.io%2fv1alpha1.Volume"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#config.kwok.x-k8s.io/v1alpha1.Component">Component</a>
, 
<a href="#config.kwok.x-k8s.io/v1alpha1.ComponentPatches">ComponentPatches</a>
</p>
<p>
<p>Volume represents a volume that is accessible to the containers running in a component.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Name of the volume specified.</p>
</td>
</tr>
<tr>
<td>
<code>readOnly</code>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Mounted read-only if true, read-write otherwise.</p>
</td>
</tr>
<tr>
<td>
<code>hostPath</code>
<em>
string
</em>
</td>
<td>
<p>HostPath represents a pre-existing file or directory on the host machine that is directly exposed to the container.</p>
</td>
</tr>
<tr>
<td>
<code>mountPath</code>
<em>
string
</em>
</td>
<td>
<p>MountPath within the container at which the volume should be mounted.</p>
</td>
</tr>
<tr>
<td>
<code>pathType</code>
<em>
<a href="#config.kwok.x-k8s.io/v1alpha1.HostPathType">
HostPathType
</a>
</em>
</td>
<td>
<p>PathType is the type of the HostPath.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.AttachConfig">
AttachConfig
<a href="#kwok.x-k8s.io%2fv1alpha1.AttachConfig"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.AttachSpec">AttachSpec</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.ClusterAttachSpec">ClusterAttachSpec</a>
</p>
<p>
<p>AttachConfig holds information how to attach.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>containers</code>
<em>
[]string
</em>
</td>
<td>
<p>Containers is list of container names.</p>
</td>
</tr>
<tr>
<td>
<code>logsFile</code>
<em>
string
</em>
</td>
<td>
<p>LogsFile is the file from which the attach starts</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.AttachSpec">
AttachSpec
<a href="#kwok.x-k8s.io%2fv1alpha1.AttachSpec"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.Attach">Attach</a>
</p>
<p>
<p>AttachSpec holds spec for attach.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>attaches</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.AttachConfig">
[]AttachConfig
</a>
</em>
</td>
<td>
<p>Attaches is a list of attaches to configure.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.AttachStatus">
AttachStatus
<a href="#kwok.x-k8s.io%2fv1alpha1.AttachStatus"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.Attach">Attach</a>
</p>
<p>
<p>AttachStatus holds status for attach</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>conditions</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Condition">
[]Condition
</a>
</em>
</td>
<td>
<p>Conditions holds conditions for attach</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ClusterAttachSpec">
ClusterAttachSpec
<a href="#kwok.x-k8s.io%2fv1alpha1.ClusterAttachSpec"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterAttach">ClusterAttach</a>
</p>
<p>
<p>ClusterAttachSpec holds spec for cluster attach.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>selector</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ObjectSelector">
ObjectSelector
</a>
</em>
</td>
<td>
<p>Selector is a selector to filter pods to configure.</p>
</td>
</tr>
<tr>
<td>
<code>attaches</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.AttachConfig">
[]AttachConfig
</a>
</em>
</td>
<td>
<p>Attaches is a list of attach configurations.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ClusterAttachStatus">
ClusterAttachStatus
<a href="#kwok.x-k8s.io%2fv1alpha1.ClusterAttachStatus"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterAttach">ClusterAttach</a>
</p>
<p>
<p>ClusterAttachStatus holds status for cluster attach</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>conditions</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Condition">
[]Condition
</a>
</em>
</td>
<td>
<p>Conditions holds conditions for cluster attach.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ClusterExecSpec">
ClusterExecSpec
<a href="#kwok.x-k8s.io%2fv1alpha1.ClusterExecSpec"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterExec">ClusterExec</a>
</p>
<p>
<p>ClusterExecSpec holds spec for cluster exec.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>selector</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ObjectSelector">
ObjectSelector
</a>
</em>
</td>
<td>
<p>Selector is a selector to filter pods to configure.</p>
</td>
</tr>
<tr>
<td>
<code>execs</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ExecTarget">
[]ExecTarget
</a>
</em>
</td>
<td>
<p>Execs is a list of exec to configure.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ClusterExecStatus">
ClusterExecStatus
<a href="#kwok.x-k8s.io%2fv1alpha1.ClusterExecStatus"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterExec">ClusterExec</a>
</p>
<p>
<p>ClusterExecStatus holds status for cluster exec</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>conditions</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Condition">
[]Condition
</a>
</em>
</td>
<td>
<p>Conditions holds conditions for cluster exec.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ClusterLogsSpec">
ClusterLogsSpec
<a href="#kwok.x-k8s.io%2fv1alpha1.ClusterLogsSpec"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterLogs">ClusterLogs</a>
</p>
<p>
<p>ClusterLogsSpec holds spec for cluster logs.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>selector</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ObjectSelector">
ObjectSelector
</a>
</em>
</td>
<td>
<p>Selector is a selector to filter pods to configure.</p>
</td>
</tr>
<tr>
<td>
<code>logs</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Log">
[]Log
</a>
</em>
</td>
<td>
<p>Forwards is a list of log configurations.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ClusterLogsStatus">
ClusterLogsStatus
<a href="#kwok.x-k8s.io%2fv1alpha1.ClusterLogsStatus"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterLogs">ClusterLogs</a>
</p>
<p>
<p>ClusterLogsStatus holds status for cluster logs</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>conditions</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Condition">
[]Condition
</a>
</em>
</td>
<td>
<p>Conditions holds conditions for cluster logs.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ClusterPortForwardSpec">
ClusterPortForwardSpec
<a href="#kwok.x-k8s.io%2fv1alpha1.ClusterPortForwardSpec"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterPortForward">ClusterPortForward</a>
</p>
<p>
<p>ClusterPortForwardSpec holds spec for cluster port forward.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>selector</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ObjectSelector">
ObjectSelector
</a>
</em>
</td>
<td>
<p>Selector is a selector to filter pods to configure.</p>
</td>
</tr>
<tr>
<td>
<code>forwards</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Forward">
[]Forward
</a>
</em>
</td>
<td>
<p>Forwards is a list of forwards to configure.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ClusterPortForwardStatus">
ClusterPortForwardStatus
<a href="#kwok.x-k8s.io%2fv1alpha1.ClusterPortForwardStatus"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterPortForward">ClusterPortForward</a>
</p>
<p>
<p>ClusterPortForwardStatus holds status for cluster port forward</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>conditions</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Condition">
[]Condition
</a>
</em>
</td>
<td>
<p>Conditions holds conditions for cluster port forward.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ClusterResourceUsageSpec">
ClusterResourceUsageSpec
<a href="#kwok.x-k8s.io%2fv1alpha1.ClusterResourceUsageSpec"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterResourceUsage">ClusterResourceUsage</a>
</p>
<p>
<p>ClusterResourceUsageSpec holds spec for cluster resource usage.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>selector</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ObjectSelector">
ObjectSelector
</a>
</em>
</td>
<td>
<p>Selector is a selector to filter pods to configure.</p>
</td>
</tr>
<tr>
<td>
<code>usages</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ResourceUsageContainer">
[]ResourceUsageContainer
</a>
</em>
</td>
<td>
<p>Usages is a list of resource usage for the pod.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ClusterResourceUsageStatus">
ClusterResourceUsageStatus
<a href="#kwok.x-k8s.io%2fv1alpha1.ClusterResourceUsageStatus"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterResourceUsage">ClusterResourceUsage</a>
</p>
<p>
<p>ClusterResourceUsageStatus holds status for cluster resource usage</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>conditions</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Condition">
[]Condition
</a>
</em>
</td>
<td>
<p>Conditions holds conditions for cluster resource usage</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.Condition">
Condition
<a href="#kwok.x-k8s.io%2fv1alpha1.Condition"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.AttachStatus">AttachStatus</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.ClusterAttachStatus">ClusterAttachStatus</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.ClusterExecStatus">ClusterExecStatus</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.ClusterLogsStatus">ClusterLogsStatus</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.ClusterPortForwardStatus">ClusterPortForwardStatus</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.ClusterResourceUsageStatus">ClusterResourceUsageStatus</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.ExecStatus">ExecStatus</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.LogsStatus">LogsStatus</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.MetricStatus">MetricStatus</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.PortForwardStatus">PortForwardStatus</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.ResourceUsageStatus">ResourceUsageStatus</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.StageStatus">StageStatus</a>
</p>
<p>
<p>Condition contains details for one aspect of the current state of this API Resource.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code>
<em>
string
</em>
</td>
<td>
<p>Type of condition in CamelCase or in foo.example.com/CamelCase.
Many .condition.type values are consistent across resources like Available, but because arbitrary conditions can be
useful (see .node.status.conditions), the ability to deconflict is important.
The regex it matches is (dns1123SubdomainFmt/)?(qualifiedNameFmt)</p>
</td>
</tr>
<tr>
<td>
<code>status</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ConditionStatus">
ConditionStatus
</a>
</em>
</td>
<td>
<p>Status of the condition</p>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>LastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.</p>
</td>
</tr>
<tr>
<td>
<code>reason</code>
<em>
string
</em>
</td>
<td>
<p>Reason contains a programmatic identifier indicating the reason for the condition&rsquo;s last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.</p>
</td>
</tr>
<tr>
<td>
<code>message</code>
<em>
string
</em>
</td>
<td>
<p>Message is a human readable message indicating details about the transition.
This may be an empty string.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ConditionStatus">
ConditionStatus
(<code>string</code> alias)
<a href="#kwok.x-k8s.io%2fv1alpha1.ConditionStatus"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.Condition">Condition</a>
</p>
<p>
<p>ConditionStatus is the status of a condition.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td><code>&#34;False&#34;</code></td>
<td><p>ConditionFalse means a resource is not in the condition.</p>
</td>
</tr>
<tr>
<td><code>&#34;True&#34;</code></td>
<td><p>ConditionTrue means a resource is in the condition.</p>
</td>
</tr>
<tr>
<td><code>&#34;Unknown&#34;</code></td>
<td><p>ConditionUnknown means kubernetes can&rsquo;t decide if a resource is in the condition or not.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.Dimension">
Dimension
(<code>string</code> alias)
<a href="#kwok.x-k8s.io%2fv1alpha1.Dimension"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.MetricConfig">MetricConfig</a>
</p>
<p>
<p>Dimension is a dimension of the metric.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td><code>&#34;container&#34;</code></td>
<td><p>DimensionContainer is a container dimension.</p>
</td>
</tr>
<tr>
<td><code>&#34;node&#34;</code></td>
<td><p>DimensionNode is a node dimension.</p>
</td>
</tr>
<tr>
<td><code>&#34;pod&#34;</code></td>
<td><p>DimensionPod is a pod dimension.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.EnvVar">
EnvVar
<a href="#kwok.x-k8s.io%2fv1alpha1.EnvVar"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.ExecTargetLocal">ExecTargetLocal</a>
</p>
<p>
<p>EnvVar represents an environment variable present in a Container.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code>
<em>
string
</em>
</td>
<td>
<p>Name of the environment variable.</p>
</td>
</tr>
<tr>
<td>
<code>value</code>
<em>
string
</em>
</td>
<td>
<p>Value of the environment variable.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ExecSpec">
ExecSpec
<a href="#kwok.x-k8s.io%2fv1alpha1.ExecSpec"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.Exec">Exec</a>
</p>
<p>
<p>ExecSpec holds spec for exec</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>execs</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ExecTarget">
[]ExecTarget
</a>
</em>
</td>
<td>
<p>Execs is a list of execs to configure.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ExecStatus">
ExecStatus
<a href="#kwok.x-k8s.io%2fv1alpha1.ExecStatus"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.Exec">Exec</a>
</p>
<p>
<p>ExecStatus holds status for exec</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>conditions</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Condition">
[]Condition
</a>
</em>
</td>
<td>
<p>Conditions holds conditions for exec</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ExecTarget">
ExecTarget
<a href="#kwok.x-k8s.io%2fv1alpha1.ExecTarget"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterExecSpec">ClusterExecSpec</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.ExecSpec">ExecSpec</a>
</p>
<p>
<p>ExecTarget holds information how to exec.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>containers</code>
<em>
[]string
</em>
</td>
<td>
<p>Containers is a list of containers to exec.
if not set, all containers will be execed.</p>
</td>
</tr>
<tr>
<td>
<code>local</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ExecTargetLocal">
ExecTargetLocal
</a>
</em>
</td>
<td>
<p>Local holds information how to exec to a local target.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ExecTargetLocal">
ExecTargetLocal
<a href="#kwok.x-k8s.io%2fv1alpha1.ExecTargetLocal"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.ExecTarget">ExecTarget</a>
</p>
<p>
<p>ExecTargetLocal holds information how to exec to a local target.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>workDir</code>
<em>
string
</em>
</td>
<td>
<p>WorkDir is the working directory to exec with.</p>
</td>
</tr>
<tr>
<td>
<code>envs</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.EnvVar">
[]EnvVar
</a>
</em>
</td>
<td>
<p>Envs is a list of environment variables to exec with.</p>
</td>
</tr>
<tr>
<td>
<code>securityContext</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.SecurityContext">
SecurityContext
</a>
</em>
</td>
<td>
<p>SecurityContext is the user context to exec.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ExpressionFromSource">
ExpressionFromSource
<a href="#kwok.x-k8s.io%2fv1alpha1.ExpressionFromSource"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.StageDelay">StageDelay</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.StageSpec">StageSpec</a>
</p>
<p>
<p>ExpressionFromSource represents a source for the value of a from.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>expressionFrom</code>
<em>
string
</em>
</td>
<td>
<p>ExpressionFrom is the expression used to get the value.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.FinalizerItem">
FinalizerItem
<a href="#kwok.x-k8s.io%2fv1alpha1.FinalizerItem"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.StageFinalizers">StageFinalizers</a>
</p>
<p>
<p>FinalizerItem  describes the one of the finalizers.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>value</code>
<em>
string
</em>
</td>
<td>
<p>Value is the value of the finalizer.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.Forward">
Forward
<a href="#kwok.x-k8s.io%2fv1alpha1.Forward"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterPortForwardSpec">ClusterPortForwardSpec</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.PortForwardSpec">PortForwardSpec</a>
</p>
<p>
<p>Forward holds information how to forward based on ports.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ports</code>
<em>
[]int32
</em>
</td>
<td>
<p>Ports is a list of ports to forward.
if not set, all ports will be forwarded.</p>
</td>
</tr>
<tr>
<td>
<code>target</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ForwardTarget">
ForwardTarget
</a>
</em>
</td>
<td>
<p>Target is the target to forward to.</p>
</td>
</tr>
<tr>
<td>
<code>command</code>
<em>
[]string
</em>
</td>
<td>
<p>Command is the command to run to forward with stdin/stdout.
if set, Target will be ignored.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ForwardTarget">
ForwardTarget
<a href="#kwok.x-k8s.io%2fv1alpha1.ForwardTarget"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.Forward">Forward</a>
</p>
<p>
<p>ForwardTarget holds information how to forward to a target.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>port</code>
<em>
int32
</em>
</td>
<td>
<p>Port is the port to forward to.</p>
</td>
</tr>
<tr>
<td>
<code>address</code>
<em>
string
</em>
</td>
<td>
<p>Address is the address to forward to.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ImpersonationConfig">
ImpersonationConfig
<a href="#kwok.x-k8s.io%2fv1alpha1.ImpersonationConfig"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.StageNext">StageNext</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.StagePatch">StagePatch</a>
</p>
<p>
<p>ImpersonationConfig describes the configuration for impersonating clients</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>username</code>
<em>
string
</em>
</td>
<td>
<p>Username the target username for the client to impersonate</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.Kind">
Kind
(<code>string</code> alias)
<a href="#kwok.x-k8s.io%2fv1alpha1.Kind"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.MetricConfig">MetricConfig</a>
</p>
<p>
<p>Kind is kind of metric configuration.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td><code>&#34;counter&#34;</code></td>
<td><p>KindCounter is a counter metric.</p>
</td>
</tr>
<tr>
<td><code>&#34;gauge&#34;</code></td>
<td><p>KindGauge is a gauge metric.</p>
</td>
</tr>
<tr>
<td><code>&#34;histogram&#34;</code></td>
<td><p>KindHistogram is a histogram metric.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.Log">
Log
<a href="#kwok.x-k8s.io%2fv1alpha1.Log"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterLogsSpec">ClusterLogsSpec</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.LogsSpec">LogsSpec</a>
</p>
<p>
<p>Log holds information how to forward logs.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>containers</code>
<em>
[]string
</em>
</td>
<td>
<p>Containers is list of container names.</p>
</td>
</tr>
<tr>
<td>
<code>logsFile</code>
<em>
string
</em>
</td>
<td>
<p>LogsFile is the file from which the log forward starts</p>
</td>
</tr>
<tr>
<td>
<code>follow</code>
<em>
bool
</em>
</td>
<td>
<p>Follow up if true</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.LogsSpec">
LogsSpec
<a href="#kwok.x-k8s.io%2fv1alpha1.LogsSpec"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.Logs">Logs</a>
</p>
<p>
<p>LogsSpec holds spec for logs.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>logs</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Log">
[]Log
</a>
</em>
</td>
<td>
<p>Logs is a list of logs to configure.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.LogsStatus">
LogsStatus
<a href="#kwok.x-k8s.io%2fv1alpha1.LogsStatus"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.Logs">Logs</a>
</p>
<p>
<p>LogsStatus holds status for logs</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>conditions</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Condition">
[]Condition
</a>
</em>
</td>
<td>
<p>Conditions holds conditions for logs</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.MetricBucket">
MetricBucket
<a href="#kwok.x-k8s.io%2fv1alpha1.MetricBucket"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.MetricConfig">MetricConfig</a>
</p>
<p>
<p>MetricBucket is a single bucket for a metric.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>le</code>
<em>
float64
</em>
</td>
<td>
<p>Le is less-than or equal.</p>
</td>
</tr>
<tr>
<td>
<code>value</code>
<em>
string
</em>
</td>
<td>
<p>Value is a CEL expression.</p>
</td>
</tr>
<tr>
<td>
<code>hidden</code>
<em>
bool
</em>
</td>
<td>
<p>Hidden is means that this bucket not shown in the metric.
but value will be calculated and cumulative into the next bucket.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.MetricConfig">
MetricConfig
<a href="#kwok.x-k8s.io%2fv1alpha1.MetricConfig"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.MetricSpec">MetricSpec</a>
</p>
<p>
<p>MetricConfig provides metric configuration to a single metric</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code>
<em>
string
</em>
</td>
<td>
<p>Name is the fully-qualified name of the metric.</p>
</td>
</tr>
<tr>
<td>
<code>help</code>
<em>
string
</em>
</td>
<td>
<p>Help provides information about this metric.</p>
</td>
</tr>
<tr>
<td>
<code>kind</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Kind">
Kind
</a>
</em>
</td>
<td>
<p>Kind is kind of metric</p>
</td>
</tr>
<tr>
<td>
<code>labels</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.MetricLabel">
[]MetricLabel
</a>
</em>
</td>
<td>
<p>Labels are metric labels.</p>
</td>
</tr>
<tr>
<td>
<code>value</code>
<em>
string
</em>
</td>
<td>
<p>Value is a CEL expression.</p>
</td>
</tr>
<tr>
<td>
<code>buckets</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.MetricBucket">
[]MetricBucket
</a>
</em>
</td>
<td>
<p>Buckets is a list of buckets for a histogram metric.</p>
</td>
</tr>
<tr>
<td>
<code>dimension</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Dimension">
Dimension
</a>
</em>
</td>
<td>
<p>Dimension is a dimension of the metric.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.MetricLabel">
MetricLabel
<a href="#kwok.x-k8s.io%2fv1alpha1.MetricLabel"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.MetricConfig">MetricConfig</a>
</p>
<p>
<p>MetricLabel holds label name and the value of the label.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code>
<em>
string
</em>
</td>
<td>
<p>Name is a label name.</p>
</td>
</tr>
<tr>
<td>
<code>value</code>
<em>
string
</em>
</td>
<td>
<p>Value is a CEL expression.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.MetricSpec">
MetricSpec
<a href="#kwok.x-k8s.io%2fv1alpha1.MetricSpec"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.Metric">Metric</a>
</p>
<p>
<p>MetricSpec holds spec for metrics.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>path</code>
<em>
string
</em>
</td>
<td>
<p>Path is a restful service path.</p>
</td>
</tr>
<tr>
<td>
<code>metrics</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.MetricConfig">
[]MetricConfig
</a>
</em>
</td>
<td>
<p>Metrics is a list of metric configurations.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.MetricStatus">
MetricStatus
<a href="#kwok.x-k8s.io%2fv1alpha1.MetricStatus"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.Metric">Metric</a>
</p>
<p>
<p>MetricStatus holds status for metrics</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>conditions</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Condition">
[]Condition
</a>
</em>
</td>
<td>
<p>Conditions holds conditions for metrics.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ObjectSelector">
ObjectSelector
<a href="#kwok.x-k8s.io%2fv1alpha1.ObjectSelector"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterAttachSpec">ClusterAttachSpec</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.ClusterExecSpec">ClusterExecSpec</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.ClusterLogsSpec">ClusterLogsSpec</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.ClusterPortForwardSpec">ClusterPortForwardSpec</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.ClusterResourceUsageSpec">ClusterResourceUsageSpec</a>
</p>
<p>
<p>ObjectSelector holds information how to match based on namespace and name.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>matchNamespaces</code>
<em>
[]string
</em>
</td>
<td>
<p>MatchNamespaces is a list of namespaces to match.
if not set, all namespaces will be matched.</p>
</td>
</tr>
<tr>
<td>
<code>matchNames</code>
<em>
[]string
</em>
</td>
<td>
<p>MatchNames is a list of names to match.
if not set, all names will be matched.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.PortForwardSpec">
PortForwardSpec
<a href="#kwok.x-k8s.io%2fv1alpha1.PortForwardSpec"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.PortForward">PortForward</a>
</p>
<p>
<p>PortForwardSpec holds spec for port forward.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>forwards</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Forward">
[]Forward
</a>
</em>
</td>
<td>
<p>Forwards is a list of forwards to configure.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.PortForwardStatus">
PortForwardStatus
<a href="#kwok.x-k8s.io%2fv1alpha1.PortForwardStatus"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.PortForward">PortForward</a>
</p>
<p>
<p>PortForwardStatus holds status for port forward</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>conditions</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Condition">
[]Condition
</a>
</em>
</td>
<td>
<p>Conditions holds conditions for port forward</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ResourceUsageContainer">
ResourceUsageContainer
<a href="#kwok.x-k8s.io%2fv1alpha1.ResourceUsageContainer"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.ClusterResourceUsageSpec">ClusterResourceUsageSpec</a>
, 
<a href="#kwok.x-k8s.io/v1alpha1.ResourceUsageSpec">ResourceUsageSpec</a>
</p>
<p>
<p>ResourceUsageContainer holds spec for resource usage container.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>containers</code>
<em>
[]string
</em>
</td>
<td>
<p>Containers is list of container names.</p>
</td>
</tr>
<tr>
<td>
<code>usage</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ResourceUsageValue">
map[string]sigs.k8s.io/kwok/pkg/apis/v1alpha1.ResourceUsageValue
</a>
</em>
</td>
<td>
<p>Usage is a list of resource usage for the container.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ResourceUsageSpec">
ResourceUsageSpec
<a href="#kwok.x-k8s.io%2fv1alpha1.ResourceUsageSpec"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.ResourceUsage">ResourceUsage</a>
</p>
<p>
<p>ResourceUsageSpec holds spec for resource usage.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>usages</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ResourceUsageContainer">
[]ResourceUsageContainer
</a>
</em>
</td>
<td>
<p>Usages is a list of resource usage for the pod.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ResourceUsageStatus">
ResourceUsageStatus
<a href="#kwok.x-k8s.io%2fv1alpha1.ResourceUsageStatus"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.ResourceUsage">ResourceUsage</a>
</p>
<p>
<p>ResourceUsageStatus holds status for resource usage</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>conditions</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Condition">
[]Condition
</a>
</em>
</td>
<td>
<p>Conditions holds conditions for resource usage</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ResourceUsageValue">
ResourceUsageValue
<a href="#kwok.x-k8s.io%2fv1alpha1.ResourceUsageValue"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.ResourceUsageContainer">ResourceUsageContainer</a>
</p>
<p>
<p>ResourceUsageValue holds value for resource usage.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>value</code>
<em>
k8s.io/apimachinery/pkg/api/resource.Quantity
</em>
</td>
<td>
<p>Value is the value for resource usage.</p>
</td>
</tr>
<tr>
<td>
<code>expression</code>
<em>
string
</em>
</td>
<td>
<p>Expression is the expression for resource usage.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.SecurityContext">
SecurityContext
<a href="#kwok.x-k8s.io%2fv1alpha1.SecurityContext"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.ExecTargetLocal">ExecTargetLocal</a>
</p>
<p>
<p>SecurityContext specifies the existing uid and gid to run exec command in container process.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>runAsUser</code>
<em>
int64
</em>
</td>
<td>
<p>RunAsUser is the existing uid to run exec command in container process.</p>
</td>
</tr>
<tr>
<td>
<code>runAsGroup</code>
<em>
int64
</em>
</td>
<td>
<p>RunAsGroup is the existing gid to run exec command in container process.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.SelectorOperator">
SelectorOperator
(<code>string</code> alias)
<a href="#kwok.x-k8s.io%2fv1alpha1.SelectorOperator"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.SelectorRequirement">SelectorRequirement</a>
</p>
<p>
<p>SelectorOperator is a label selector operator is the set of operators that can be used in a selector requirement.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td><code>&#34;DoesNotExist&#34;</code></td>
<td><p>SelectorOpDoesNotExist is the negated existence operator.</p>
</td>
</tr>
<tr>
<td><code>&#34;Exists&#34;</code></td>
<td><p>SelectorOpExists is the existence operator.</p>
</td>
</tr>
<tr>
<td><code>&#34;In&#34;</code></td>
<td><p>SelectorOpIn is the set inclusion operator.</p>
</td>
</tr>
<tr>
<td><code>&#34;NotIn&#34;</code></td>
<td><p>SelectorOpNotIn is the negated set inclusion operator.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.SelectorRequirement">
SelectorRequirement
<a href="#kwok.x-k8s.io%2fv1alpha1.SelectorRequirement"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.StageSelector">StageSelector</a>
</p>
<p>
<p>SelectorRequirement is a resource selector requirement is a selector that contains values, a key,
and an operator that relates the key and values.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>key</code>
<em>
string
</em>
</td>
<td>
<p>The name of the scope that the selector applies to.</p>
</td>
</tr>
<tr>
<td>
<code>operator</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.SelectorOperator">
SelectorOperator
</a>
</em>
</td>
<td>
<p>Represents a scope&rsquo;s relationship to a set of values.</p>
</td>
</tr>
<tr>
<td>
<code>values</code>
<em>
[]string
</em>
</td>
<td>
<p>An array of string values.
If the operator is In, NotIn, Intersection or NotIntersection, the values array must be non-empty.
If the operator is Exists or DoesNotExist, the values array must be empty.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.StageDelay">
StageDelay
<a href="#kwok.x-k8s.io%2fv1alpha1.StageDelay"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.StageSpec">StageSpec</a>
</p>
<p>
<p>StageDelay describes the delay time before going to next.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>durationMilliseconds</code>
<em>
int64
</em>
</td>
<td>
<p>DurationMilliseconds indicates the stage delay time.
If JitterDurationMilliseconds is less than DurationMilliseconds, then JitterDurationMilliseconds is used.</p>
</td>
</tr>
<tr>
<td>
<code>durationFrom</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ExpressionFromSource">
ExpressionFromSource
</a>
</em>
</td>
<td>
<p>DurationFrom is the expression used to get the value.
If it is a time.Time type, getting the value will be minus time.Now() to get DurationMilliseconds
If it is a string type, the value get will be parsed by time.ParseDuration.</p>
</td>
</tr>
<tr>
<td>
<code>jitterDurationMilliseconds</code>
<em>
int64
</em>
</td>
<td>
<p>JitterDurationMilliseconds is the duration plus an additional amount chosen uniformly
at random from the interval between DurationMilliseconds and JitterDurationMilliseconds.</p>
</td>
</tr>
<tr>
<td>
<code>jitterDurationFrom</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ExpressionFromSource">
ExpressionFromSource
</a>
</em>
</td>
<td>
<p>JitterDurationFrom is the expression used to get the value.
If it is a time.Time type, getting the value will be minus time.Now() to get JitterDurationMilliseconds
If it is a string type, the value get will be parsed by time.ParseDuration.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.StageEvent">
StageEvent
<a href="#kwok.x-k8s.io%2fv1alpha1.StageEvent"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.StageNext">StageNext</a>
</p>
<p>
<p>StageEvent describes one event in the Kubernetes.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code>
<em>
string
</em>
</td>
<td>
<p>Type is the type of this event (Normal, Warning), It is machine-readable.</p>
</td>
</tr>
<tr>
<td>
<code>reason</code>
<em>
string
</em>
</td>
<td>
<p>Reason is why the action was taken. It is human-readable.</p>
</td>
</tr>
<tr>
<td>
<code>message</code>
<em>
string
</em>
</td>
<td>
<p>Message is a human-readable description of the status of this operation.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.StageFinalizers">
StageFinalizers
<a href="#kwok.x-k8s.io%2fv1alpha1.StageFinalizers"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.StageNext">StageNext</a>
</p>
<p>
<p>StageFinalizers describes the modifications in the finalizers of a resource.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>add</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.FinalizerItem">
[]FinalizerItem
</a>
</em>
</td>
<td>
<p>Add means that the Finalizers will be added to the resource.</p>
</td>
</tr>
<tr>
<td>
<code>remove</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.FinalizerItem">
[]FinalizerItem
</a>
</em>
</td>
<td>
<p>Remove means that the Finalizers will be removed from the resource.</p>
</td>
</tr>
<tr>
<td>
<code>empty</code>
<em>
bool
</em>
</td>
<td>
<p>Empty means that the Finalizers for that resource will be emptied.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.StageNext">
StageNext
<a href="#kwok.x-k8s.io%2fv1alpha1.StageNext"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.StageSpec">StageSpec</a>
</p>
<p>
<p>StageNext describes a stage will be moved to.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>event</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.StageEvent">
StageEvent
</a>
</em>
</td>
<td>
<p>Event means that an event will be sent.</p>
</td>
</tr>
<tr>
<td>
<code>finalizers</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.StageFinalizers">
StageFinalizers
</a>
</em>
</td>
<td>
<p>Finalizers means that finalizers will be modified.</p>
</td>
</tr>
<tr>
<td>
<code>delete</code>
<em>
bool
</em>
</td>
<td>
<p>Delete means that the resource will be deleted if true.</p>
</td>
</tr>
<tr>
<td>
<code>patches</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.StagePatch">
[]StagePatch
</a>
</em>
</td>
<td>
<p>Patches means that the resource will be patched.</p>
</td>
</tr>
<tr>
<td>
<code>statusTemplate</code>
<em>
string
</em>
</td>
<td>
<p>StatusTemplate indicates the template for modifying the status of the resource in the next.
Deprecated: Use Patches instead.</p>
</td>
</tr>
<tr>
<td>
<code>statusSubresource</code>
<em>
string
</em>
</td>
<td>
<p>StatusSubresource indicates the name of the subresource that will be patched. The support for
this field is not available in Pod and Node resources.
Deprecated: Use Patches instead.</p>
</td>
</tr>
<tr>
<td>
<code>statusPatchAs</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ImpersonationConfig">
ImpersonationConfig
</a>
</em>
</td>
<td>
<p>StatusPatchAs indicates the impersonating configuration for client when patching status.
In most cases this will be empty, in which case the default client service account will be used.
When this is not empty, a corresponding rbac change is required to grant <code>impersonate</code> privilege.
The support for this field is not available in Pod and Node resources.
Deprecated: Use Patches instead.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.StagePatch">
StagePatch
<a href="#kwok.x-k8s.io%2fv1alpha1.StagePatch"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.StageNext">StageNext</a>
</p>
<p>
<p>StagePatch describes the patch for the resource.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>subresource</code>
<em>
string
</em>
</td>
<td>
<p>Subresource indicates the name of the subresource that will be patched.</p>
</td>
</tr>
<tr>
<td>
<code>root</code>
<em>
string
</em>
</td>
<td>
<p>Root indicates the root of the template calculated by the patch.</p>
</td>
</tr>
<tr>
<td>
<code>template</code>
<em>
string
</em>
</td>
<td>
<p>Template indicates the template for modifying the resource in the next.</p>
</td>
</tr>
<tr>
<td>
<code>type</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.StagePatchType">
StagePatchType
</a>
</em>
</td>
<td>
<p>Type indicates the type of the patch.</p>
</td>
</tr>
<tr>
<td>
<code>impersonation</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ImpersonationConfig">
ImpersonationConfig
</a>
</em>
</td>
<td>
<p>Impersonation indicates the impersonating configuration for client when patching status.
In most cases this will be empty, in which case the default client service account will be used.
When this is not empty, a corresponding rbac change is required to grant <code>impersonate</code> privilege.
The support for this field is not available in Pod and Node resources.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.StagePatchType">
StagePatchType
(<code>string</code> alias)
<a href="#kwok.x-k8s.io%2fv1alpha1.StagePatchType"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.StagePatch">StagePatch</a>
</p>
<p>
<p>StagePatchType is the type of the patch.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td><code>&#34;json&#34;</code></td>
<td><p>StagePatchTypeJSONPatch is the JSON patch type.</p>
</td>
</tr>
<tr>
<td><code>&#34;merge&#34;</code></td>
<td><p>StagePatchTypeMergePatch is the merge patch type.</p>
</td>
</tr>
<tr>
<td><code>&#34;strategic&#34;</code></td>
<td><p>StagePatchTypeStrategicMergePatch is the strategic merge patch type.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.StageResourceRef">
StageResourceRef
<a href="#kwok.x-k8s.io%2fv1alpha1.StageResourceRef"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.StageSpec">StageSpec</a>
</p>
<p>
<p>StageResourceRef specifies the kind and version of the resource.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiGroup</code>
<em>
string
</em>
</td>
<td>
<p>APIGroup of the referent.</p>
</td>
</tr>
<tr>
<td>
<code>kind</code>
<em>
string
</em>
</td>
<td>
<p>Kind of the referent.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.StageSelector">
StageSelector
<a href="#kwok.x-k8s.io%2fv1alpha1.StageSelector"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.StageSpec">StageSpec</a>
</p>
<p>
<p>StageSelector is a resource selector. the result of matchLabels and matchAnnotations and
matchExpressions are ANDed. An empty resource selector matches all objects. A null
resource selector matches no objects.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>matchLabels</code>
<em>
map[string]string
</em>
</td>
<td>
<p>MatchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is &ldquo;.metadata.labels[key]&rdquo;, the
operator is &ldquo;In&rdquo;, and the values array contains only &ldquo;value&rdquo;. The requirements are ANDed.</p>
</td>
</tr>
<tr>
<td>
<code>matchAnnotations</code>
<em>
map[string]string
</em>
</td>
<td>
<p>MatchAnnotations is a map of {key,value} pairs. A single {key,value} in the matchAnnotations
map is equivalent to an element of matchExpressions, whose key field is &ldquo;.metadata.annotations[key]&rdquo;, the
operator is &ldquo;In&rdquo;, and the values array contains only &ldquo;value&rdquo;. The requirements are ANDed.</p>
</td>
</tr>
<tr>
<td>
<code>matchExpressions</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.SelectorRequirement">
[]SelectorRequirement
</a>
</em>
</td>
<td>
<p>MatchExpressions is a list of label selector requirements. The requirements are ANDed.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.StageSpec">
StageSpec
<a href="#kwok.x-k8s.io%2fv1alpha1.StageSpec"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.Stage">Stage</a>
</p>
<p>
<p>StageSpec defines the specification for Stage.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>resourceRef</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.StageResourceRef">
StageResourceRef
</a>
</em>
</td>
<td>
<p>ResourceRef specifies the Kind and version of the resource.</p>
</td>
</tr>
<tr>
<td>
<code>selector</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.StageSelector">
StageSelector
</a>
</em>
</td>
<td>
<p>Selector specifies the stags will be applied to the selected resource.</p>
</td>
</tr>
<tr>
<td>
<code>weight</code>
<em>
int
</em>
</td>
<td>
<p>Weight means when multiple stages share the same ResourceRef and Selector,
a random stage will be matched as the next stage based on the weight.</p>
</td>
</tr>
<tr>
<td>
<code>weightFrom</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.ExpressionFromSource">
ExpressionFromSource
</a>
</em>
</td>
<td>
<p>WeightFrom means is the expression used to get the value.
If it is a number type, convert to int.
If it is a string type, the value get will be parsed by strconv.ParseInt.</p>
</td>
</tr>
<tr>
<td>
<code>delay</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.StageDelay">
StageDelay
</a>
</em>
</td>
<td>
<p>Delay means there is a delay in this stage.</p>
</td>
</tr>
<tr>
<td>
<code>next</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.StageNext">
StageNext
</a>
</em>
</td>
<td>
<p>Next indicates that this stage will be moved to.</p>
</td>
</tr>
<tr>
<td>
<code>immediateNextStage</code>
<em>
bool
</em>
</td>
<td>
<p>ImmediateNextStage means that the next stage of matching is performed immediately, without waiting for the Apiserver to push.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.StageStatus">
StageStatus
<a href="#kwok.x-k8s.io%2fv1alpha1.StageStatus"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.Stage">Stage</a>
</p>
<p>
<p>StageStatus holds status for the Stage</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>conditions</code>
<em>
<a href="#kwok.x-k8s.io/v1alpha1.Condition">
[]Condition
</a>
</em>
</td>
<td>
<p>Conditions holds conditions for the Stage.</p>
</td>
</tr>
</tbody>
</table>
