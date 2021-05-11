---
title: API Documentation
linktitle: API Documentation
description: Reference of the jx-promote configuration
weight: 10
---
<p>Packages:</p>
<ul>
<li>
<a href="#kubernetes-client.io%2fv1">kubernetes-client.io/v1</a>
</li>
</ul>
<h2 id="kubernetes-client.io/v1">kubernetes-client.io/v1</h2>
<p>
<p>Package v1alpha1 is the v1alpha1 version of the API.</p>
</p>
Resource Types:
<ul><li>
<a href="#kubernetes-client.io/v1.ExternalSecret">ExternalSecret</a>
</li></ul>
<h3 id="kubernetes-client.io/v1.ExternalSecret">ExternalSecret
</h3>
<p>
<p>ExternalSecret represents a collection of mappings of Secrets to destinations in the underlying secret store (e.g. Vault keys)</p>
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
<code>apiVersion</code></br>
string</td>
<td>
<code>
kubernetes-client.io/v1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>ExternalSecret</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#kubernetes-client.io/v1.ExternalSecretSpec">
ExternalSecretSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Spec holds the desired state of the ExternalSecret from the client</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>backendType</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>vaultMountPoint</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>vaultRole</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>projectId</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>keyVaultName</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>data</code></br>
<em>
<a href="#kubernetes-client.io/v1.Data">
[]Data
</a>
</em>
</td>
<td>
<p>Data the data for each entry in the Secret</p>
</td>
</tr>
<tr>
<td>
<code>template</code></br>
<em>
<a href="#kubernetes-client.io/v1.Template">
Template
</a>
</em>
</td>
<td>
<p>Template</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#kubernetes-client.io/v1.ExternalSecretStatus">
ExternalSecretStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Status holds the current status</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kubernetes-client.io/v1.Data">Data
</h3>
<p>
(<em>Appears on:</em>
<a href="#kubernetes-client.io/v1.ExternalSecretSpec">ExternalSecretSpec</a>)
</p>
<p>
<p>Data the data properties</p>
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
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name name of the secret data entry</p>
</td>
</tr>
<tr>
<td>
<code>key</code></br>
<em>
string
</em>
</td>
<td>
<p>Key the key in the underlying secret storage (e.g. the key in vault)</p>
</td>
</tr>
<tr>
<td>
<code>property</code></br>
<em>
string
</em>
</td>
<td>
<p>Property the property in the underlying secret storage (e.g.  in vault)</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kubernetes-client.io/v1.ExternalSecretSpec">ExternalSecretSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#kubernetes-client.io/v1.ExternalSecret">ExternalSecret</a>)
</p>
<p>
<p>ExternalSecretSpec defines the desired state of ExternalSecret.</p>
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
<code>backendType</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>vaultMountPoint</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>vaultRole</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>projectId</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>keyVaultName</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>data</code></br>
<em>
<a href="#kubernetes-client.io/v1.Data">
[]Data
</a>
</em>
</td>
<td>
<p>Data the data for each entry in the Secret</p>
</td>
</tr>
<tr>
<td>
<code>template</code></br>
<em>
<a href="#kubernetes-client.io/v1.Template">
Template
</a>
</em>
</td>
<td>
<p>Template</p>
</td>
</tr>
</tbody>
</table>
<h3 id="kubernetes-client.io/v1.ExternalSecretStatus">ExternalSecretStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#kubernetes-client.io/v1.ExternalSecret">ExternalSecret</a>)
</p>
<p>
<p>ExternalSecretStatus defines the current status of the ExternalSecret.</p>
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
<code>lastSync</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>observedGeneration</code></br>
<em>
int
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="kubernetes-client.io/v1.Template">Template
</h3>
<p>
(<em>Appears on:</em>
<a href="#kubernetes-client.io/v1.ExternalSecretSpec">ExternalSecretSpec</a>)
</p>
<p>
<p>Template the template data</p>
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
<code>type</code></br>
<em>
string
</em>
</td>
<td>
<p>Type the type of the secret</p>
</td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<p>Metadata the metadata such as labels or annotations</p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>
on git commit <code>f867b89</code>.
</em></p>
