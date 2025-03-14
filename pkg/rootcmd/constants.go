package rootcmd

import "os"

var (
	// TopLevelCommand top level command
	TopLevelCommand = "jx-secret"

	// BinaryName the name of the command binary in help
	BinaryName = "jx-secret"
)

func init() {
	binaryName, ok := os.LookupEnv("BINARY_NAME")
	if ok {
		BinaryName = binaryName
	}
	topLevelCommand, ok := os.LookupEnv("TOP_LEVEL_COMMAND")
	if ok {
		TopLevelCommand = topLevelCommand
	}
}
