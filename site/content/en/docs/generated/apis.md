---
title: API reference
bookToc: false
---
<h1>API reference</h1>
<p>Packages:</p>
<ul>
<li>
<a href="#config.kwok.x-k8s.io%2fv1alpha1">config.kwok.x-k8s.io/v1alpha1</a>
</li>
<li>
<a href="#kwok.x-k8s.io%2fv1alpha1">kwok.x-k8s.io/v1alpha1</a>
</li>
</ul>
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
<a href="#config.kwok.x-k8s.io/v1alpha1.KwokctlConfiguration">KwokctlConfiguration</a>
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
<a href="#kwok.x-k8s.io/v1alpha1.Exec">Exec</a>
</li>
<li>
<a href="#kwok.x-k8s.io/v1alpha1.Logs">Logs</a>
</li>
<li>
<a href="#kwok.x-k8s.io/v1alpha1.PortForward">PortForward</a>
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
<p>Weight means the current stage, in case of multiple stages,
a random stage will be matched as the next stage based on the weight.</p>
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
</tbody>
</table>
<h2 id="references">
References
<a href="#references"> #</a>
</h2>
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
</tbody>
</table>
<h3 id="config.kwok.x-k8s.io/v1alpha1.Env">
Env
<a href="#config.kwok.x-k8s.io%2fv1alpha1.Env"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#config.kwok.x-k8s.io/v1alpha1.Component">Component</a>
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
<code>manageAllNodes</code>
<em>
bool
</em>
</td>
<td>
<p>Default option to manage (i.e., maintain heartbeat/liveness of) all Nodes or not.
is the default value for flag &ndash;manage-all-nodes</p>
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
Note: when <code>all-node-manage</code> is specified as true, this is a no-op.
is the default value for flag &ndash;manage-nodes-with-annotation-selector</p>
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
Note: when <code>all-node-manage</code> is specified as true, this is a no-op.
is the default value for flag &ndash;manage-nodes-with-label-selector</p>
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
is the default value for flag &ndash;disregard-status-with-annotation-selector</p>
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
is the default value for flag &ndash;disregard-status-with-label-selector</p>
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
is the default value for flag &ndash;experimental-enable-cni</p>
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
<code>dockerComposeVersion</code>
<em>
string
</em>
</td>
<td>
<p>DockerComposeVersion is the version of docker-compose to use.
is the default value for env KWOK_DOCKER_COMPOSE_VERSION
Deprecated: docker compose will be removed in a future release</p>
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
<code>disableKubeScheduler</code>
<em>
bool
</em>
</td>
<td>
<p>DisableKubeScheduler is the flag to disable kube-scheduler.
is the default value for flag &ndash;disable-kube-scheduler and env KWOK_DISABLE_KUBE_SCHEDULER</p>
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
is the default value for flag &ndash;disable-kube-controller-manager and env KWOK_DISABLE_KUBE_CONTROLLER_MANAGER</p>
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
is the default value for env KWOK_ETCD_BINARY_TAR</p>
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
is the default value for env KWOK_PROMETHEUS_BINARY_TAR</p>
</td>
</tr>
<tr>
<td>
<code>dockerComposeBinaryPrefix</code>
<em>
string
</em>
</td>
<td>
<p>DockerComposeBinaryPrefix is the binary of docker-compose.
is the default value for env KWOK_DOCKER_COMPOSE_BINARY_PREFIX
Deprecated: docker compose will be removed in a future release</p>
</td>
</tr>
<tr>
<td>
<code>dockerComposeBinary</code>
<em>
string
</em>
</td>
<td>
<p>DockerComposeBinary is the binary of Docker compose.
is the default value for flag &ndash;docker-compose-binary and env KWOK_DOCKER_COMPOSE_BINARY
Deprecated: docker compose will be removed in a future release</p>
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
is the default value for env KWOK_MODE</p>
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
</tbody>
</table>
<h3 id="kwok.x-k8s.io/v1alpha1.ExpressionFromSource">
ExpressionFromSource
<a href="#kwok.x-k8s.io%2fv1alpha1.ExpressionFromSource"> #</a>
</h3>
<p>
<em>Appears on: </em>
<a href="#kwok.x-k8s.io/v1alpha1.StageDelay">StageDelay</a>
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
<code>statusTemplate</code>
<em>
string
</em>
</td>
<td>
<p>StatusTemplate indicates the template for modifying the status of the resource in the next.</p>
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
<p>Weight means the current stage, in case of multiple stages,
a random stage will be matched as the next stage based on the weight.</p>
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
