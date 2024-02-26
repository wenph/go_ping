package show

import (
	"fmt"
	"os"
)

var (
	version = "go_ping-v1.1.3"
)

func ShowOrUpgradeVersion(showVersion bool) {
	if showVersion {
		fmt.Println(fmt.Sprintf("Current Version:%s", version))
		os.Exit(0)
	}
}
