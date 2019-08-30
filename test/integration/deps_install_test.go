// +build integration

package integration_test

import (
	"fmt"
	"log"
	"os/exec"
)

// Check if mercurial client is installed. If not then try to install.
//
// CURRENTLY SUPPORTED ONLY APT-GET INSTALLER to satisfy CI pipelines.
func EnsureHgInstalled() {
	if err := foundExecutable("hg"); err != nil {
		log.Println("hg executable not found in $PATH. Try to install with apt-get")

		err = foundExecutable("apt-get")
		panicOnError(err, "while checking if apt-get executable exists")

		err = exec.Command("apt-get", "--assume-yes", "install", "mercurial").Run()
		panicOnError(err, "while installing mercurial via apt-get")
	}
}

func panicOnError(err error, context string) {
	if err != nil {
		panic(fmt.Sprintf("%s: %v", context, err))
	}
}

func foundExecutable(name string) error {
	_, err := exec.LookPath(name)
	return err
}
