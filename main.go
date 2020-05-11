package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
)

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func debug(str string) {
	if os.Getenv("TUPLECTL_DEBUG") != "" {
		log.Print(str)
	}
}

func warn(str string) {
	warn := "WARNING: "
	shouldPrintColor := runtime.GOOS == "darwin"
	if shouldPrintColor {
		// highlight WARNING in red if we're on a mac
		fmt.Println("\033[31m" + warn + "\033[30m" + str)
	} else {
		fmt.Println(warn + str)
	}
}

func version() string {
	return "AUTOREPLACED-VERSION"
}

func usage() {
	fmt.Println("TODO usage")
	os.Exit(1)
}

func status() {
	fmt.Println(getResource("status"))
}

func dispatchGet(resource string, args []string) {
	auth()
	switch resource {
	case "logstreams":
		fmt.Println("TODO logstreams")
	case "status":
		status()
	default:
		log.Panic("Unknown subcommand")
	}
}

func main() {
	// degenerate case
	if len(os.Args) < 2 {
		usage()
	}

	switch os.Args[1] {
	case "setup":
		fmt.Println("this is the critical path")
		// 1. authenticate / sign up
		auth()
		// 2. check for any deployments / logstreams
		// 3. if none, ask if k8s / lambda
		// 4. validate environment
		// 5. if failure, open docs web page
		// 6. if env is good, prompt for deployment name
		// 7. prompt for confirmation
		// 8. tail all streams for response
		// 9. success message, link to docs
	case "get":
		dispatchGet(os.Args[2], os.Args[3:])
	case "version":
		fmt.Println("Tuplectl " + version())
	default:
		usage()
	}
}
