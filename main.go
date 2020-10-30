//go:generate top

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"

	"github.com/tuplestream/hawkeye-client"
)

var Version string
var Commit string
var BuildDate string

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func getEnvOrDefault(envVar string, fallback string) string {
	val := os.Getenv(envVar)
	if val == "" {
		val = fallback
	}
	return val
}

func debug(str string) {
	if os.Getenv("TUPLECTL_DEBUG") != "" {
		log.Print(str)
	}
}

func red(str string) string {
	shouldPrintColor := runtime.GOOS == "darwin"
	if shouldPrintColor {
		// highlight in red if we're on a mac
		return "\033[31m" + str + "\033[0m"
	}
	return str
}

func warn(str string) {
	fmt.Println(red("WARNING: ") + str)
}

func version() string {
	return "Version: " + Version + " | Commit: " + Commit + " | Built: " + BuildDate
}

func usage() {
	fmt.Println("TODO usage")
	os.Exit(1)
}

func status() {
	fmt.Println(getResource("status"))
}

func dispatchGet(resource string, args []string) {
	doAuth()
	switch resource {
	case "logstreams":
		fmt.Println("TODO logstreams")
	case "status":
		status()
	default:
		log.Panic("Unknown subcommand")
	}
}

func echoData() {
	doAuth()
	if len(os.Args) <= 2 {
		fmt.Println("Usage: tuplectl echo [filename] [-]")
		os.Exit(1)
	}

	fileName := os.Args[2]
	var fd *os.File
	if os.Args[2] == "-" {
		fileName = "STDIN"
		fd = os.Stdin
	} else {
		fd, _ = os.Open(fileName)
	}

	conn, writer := hawkeye.InitiateConnection(fileName, accessToken)
	defer conn.Close()

	bytesTotal, _ := io.Copy(writer, fd)
	writer.WriteString("\n")
	writer.Flush()

	fmt.Println(fmt.Sprintf("Successfully sent %d bytes of data", bytesTotal+1))
}

func main() {
	// degenerate case
	if len(os.Args) < 2 {
		usage()
	}

	switch os.Args[1] {
	case "setup":
		// 1. authenticate / sign up
		doAuth()
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
	case "echo":
		echoData()
	case "version":
		fmt.Println("tuplectl " + version())
	default:
		usage()
	}
}
