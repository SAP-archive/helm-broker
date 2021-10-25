//go:build integration
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

		err = exec.Command("apt-get", "update").Run()
		panicOnError(err, "while update apt-get")

		err = exec.Command("apt-get", "--assume-yes", "install", "mercurial").Run()
		panicOnError(err, "while installing mercurial via apt-get")
	}
}

// Check if minio server is installed. If not then try to install.
//
// CURRENTLY SUPPORTED ONLY APT-GET INSTALLER to satisfy CI pipelines.
func EnsureMinioInstalled() {
	if err := foundExecutable("minio"); err != nil {
		log.Println("minio executable not found in $PATH. Try to download with wget")

		err = foundExecutable("wget")
		panicOnError(err, "while checking if wget executable exists")

		err = exec.Command("wget", "https://dl.min.io/server/minio/release/linux-amd64/minio").Run()
		panicOnError(err, "while download minio via wget")

		err = exec.Command("chmod", "+x", "minio").Run()
		panicOnError(err, "while change access permission for minio binary")

		err = exec.Command("mv", "minio", "/usr/local/bin/").Run()
		panicOnError(err, "while moving minio to /usr/local/bin/ path")
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
