package extsecrets

const (
	// SchemaObjectAnnotation the annotation which contains the JSON encoded schema object definition
	SchemaObjectAnnotation = "secret.jenkins-x.io/schema-object"

	// ReplicateAnnotation the annotation which lists the namespaces to replicate a Secret to when using local secrets
	ReplicateAnnotation = "secret.jenkins-x.io/replicate"
)
