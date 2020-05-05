package main

import (
	"fmt"
	"log"
	"os"
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

func version() string {
	return "x.y.z"
}

func usage() {
	fmt.Println("TODO usage")
	os.Exit(1)
}

func status() {
	fmt.Println(getResource("status"))
}

func dispatchGet(resource string, args []string) {
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

	auth()

	switch os.Args[1] {
	case "get":
		dispatchGet(os.Args[2], os.Args[3:])
	case "version":
		fmt.Println("Tuplectl version " + version())
	default:
		usage()
	}
}
