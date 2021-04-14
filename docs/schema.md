---
title: API Documentation
linktitle: API Documentation
description: Reference of the jx-promote configuration
weight: 10
---
<p>Packages:</p>
<ul>
<li>
<a href="#secret.jenkins-x.io%2fv1alpha1">secret.jenkins-x.io/v1alpha1</a>
</li>
</ul>
<h2 id="secret.jenkins-x.io/v1alpha1">secret.jenkins-x.io/v1alpha1</h2>
<p>
<p>Package v1alpha1 is the v1alpha1 version of the API.</p>
</p>
Resource Types:
<ul><li>
<a href="#secret.jenkins-x.io/v1alpha1.Schema">Schema</a>
</li></ul>
<h3 id="secret.jenkins-x.io/v1alpha1.Schema">Schema
</h3>
<p>
<p>Schema defines a schema of objects with properties</p>
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
secret.jenkins-x.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>Schema</code></td>
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
<code>Spec</code></br>
<em>
<a href="#secret.jenkins-x.io/v1alpha1.SchemaSpec">
SchemaSpec
</a>
</em>
</td>
<td>
<p>Spec the schema specification</p>
</td>
</tr>
</tbody>
</table>
<h3 id="secret.jenkins-x.io/v1alpha1.Object">Object
</h3>
<p>
(<em>Appears on:</em>
<a href="#secret.jenkins-x.io/v1alpha1.SchemaSpec">SchemaSpec</a>)
</p>
<p>
<p>Object defines a type of object with some properties</p>
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
<p>Name the name of the object kind</p>
</td>
</tr>
<tr>
<td>
<code>properties</code></br>
<em>
<a href="#secret.jenkins-x.io/v1alpha1.Property">
[]Property
</a>
</em>
</td>
<td>
<p>Properties the property definitions</p>
</td>
</tr>
<tr>
<td>
<code>mandatory</code></br>
<em>
bool
</em>
</td>
<td>
<p>Mandatory marks this secret as being mandatory to be setup before we can install a cluster</p>
</td>
</tr>
</tbody>
</table>
<h3 id="secret.jenkins-x.io/v1alpha1.Property">Property
</h3>
<p>
(<em>Appears on:</em>
<a href="#secret.jenkins-x.io/v1alpha1.Object">Object</a>)
</p>
<p>
<p>Property defines a property in an object</p>
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
<p>Name the name of the property</p>
</td>
</tr>
<tr>
<td>
<code>question</code></br>
<em>
string
</em>
</td>
<td>
<p>Question the main prompt generated in a user interface when asking to populate the property</p>
</td>
</tr>
<tr>
<td>
<code>help</code></br>
<em>
string
</em>
</td>
<td>
<p>Help the tooltip or help text for this property</p>
</td>
</tr>
<tr>
<td>
<code>defaultValue</code></br>
<em>
string
</em>
</td>
<td>
<p>DefaultValue is used to specify default values populated on startup</p>
</td>
</tr>
<tr>
<td>
<code>pattern</code></br>
<em>
string
</em>
</td>
<td>
<p>Pattern is a regular expression pattern used for validation</p>
</td>
</tr>
<tr>
<td>
<code>requires</code></br>
<em>
string
</em>
</td>
<td>
<p>Requires specifies a requirements expression</p>
</td>
</tr>
<tr>
<td>
<code>format</code></br>
<em>
string
</em>
</td>
<td>
<p>Format the format of the value</p>
</td>
</tr>
<tr>
<td>
<code>generator</code></br>
<em>
string
</em>
</td>
<td>
<p>Generator the name of the generator to use to create values
if this value is non zero we assume Generate is effectively true</p>
</td>
</tr>
<tr>
<td>
<code>template</code></br>
<em>
string
</em>
</td>
<td>
<p>Template the go template used to generate the value of this secret
if we need to combine multiple secret values together into a composite secret value.</p>
<p>For example if we want to create a maven-settings.xml file or a docker config JSON
document made up of lots of static text but some real secrets embedded we can
define the template in the schema</p>
</td>
</tr>
<tr>
<td>
<code>onlyTemplateIfBlank</code></br>
<em>
bool
</em>
</td>
<td>
<p>OnlyTemplateIfBlank if this is true then lets only regenerate a template value if the current value is empty</p>
</td>
</tr>
<tr>
<td>
<code>retry</code></br>
<em>
bool
</em>
</td>
<td>
<p>Retry enable a retry loop if a template does not evaluate correctly first time</p>
</td>
</tr>
<tr>
<td>
<code>labels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<p>Labels allows arbitrary metadata labels to be associated with the property</p>
</td>
</tr>
<tr>
<td>
<code>minLength</code></br>
<em>
int
</em>
</td>
<td>
<p>MinLength the minimum number of characters in the value</p>
</td>
</tr>
<tr>
<td>
<code>maxLength</code></br>
<em>
int
</em>
</td>
<td>
<p>MaxLength the maximum number of characters in the value</p>
</td>
</tr>
<tr>
<td>
<code>noMask</code></br>
<em>
bool
</em>
</td>
<td>
<p>NoMask whether to exclude from Secret masking in logs</p>
</td>
</tr>
</tbody>
</table>
<h3 id="secret.jenkins-x.io/v1alpha1.SchemaSpec">SchemaSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#secret.jenkins-x.io/v1alpha1.Schema">Schema</a>)
</p>
<p>
<p>SchemaSpec defines the objects and their properties</p>
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
<code>Objects</code></br>
<em>
<a href="#secret.jenkins-x.io/v1alpha1.Object">
[]Object
</a>
</em>
</td>
<td>
<p>Objects the list of objects (or kinds) in the schema</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>
on git commit <code>8470324</code>.
</em></p>
