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
<ul></ul>
<h3 id="secret.jenkins-x.io/v1alpha1.Object">Object
</h3>
<p>
(<em>Appears on:</em>
<a href="#secret.jenkins-x.io/v1alpha1.Spec">Spec</a>)
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
<code>Name</code></br>
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
<code>Properties</code></br>
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
<code>Name</code></br>
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
<code>Question</code></br>
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
<code>Help</code></br>
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
<code>DefaultValue</code></br>
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
<code>Pattern</code></br>
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
<code>Requires</code></br>
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
<code>Format</code></br>
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
<code>Generator</code></br>
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
<code>Labels</code></br>
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
<code>MinLength</code></br>
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
<code>MaxLength</code></br>
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
<code>Mask</code></br>
<em>
bool
</em>
</td>
<td>
<p>Mask whether a mask is used on input</p>
</td>
</tr>
</tbody>
</table>
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
<code>APIVersion</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>Kind</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>Spec</code></br>
<em>
<a href="#secret.jenkins-x.io/v1alpha1.Spec">
Spec
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="secret.jenkins-x.io/v1alpha1.Spec">Spec
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
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>
on git commit <code>ff815cf</code>.
</em></p>
