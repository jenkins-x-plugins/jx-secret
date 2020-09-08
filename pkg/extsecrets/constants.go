package extsecrets

const (
	// SchemaObjectAnnotation the annotation which contains the JSON encoded schema object definition
	SchemaObjectAnnotation = "secret.jenkins-x.io/schema-object"

	// ReplicateToAnnotation the annotation which lists the namespaces to replicate a Secret to when using local secrets
	ReplicateToAnnotation = "secret.jenkins-x.io/replicate-to"

	// ReplicaAnnotation the annotation on an ExternalSecret which is a replica
	ReplicaAnnotation = "secret.jenkins-x.io/replica"
)
