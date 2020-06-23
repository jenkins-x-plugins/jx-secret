package input

type Interface interface {
	PickPassword(message string, help string) (string, error)
}
