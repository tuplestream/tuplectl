//go:generate top

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"

	"github.com/alecthomas/kong"
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

func canPrettyPrint() bool {
	return runtime.GOOS == "darwin"
}

func bold(str string) string {
	if canPrettyPrint() {
		return "\033[1m" + str + "\033[0m"
	}
	return str
}

func red(str string) string {
	if canPrettyPrint() {
		// highlight in red if we're on a mac
		return "\033[31m" + str + "\033[0m"
	}
	return str
}

func warn(str string) {
	fmt.Println(red("WARNING: ") + str)
}

func oddChar(str string) string {
	if canPrettyPrint() {
		return str
	}
	return ""
}

func version() string {
	return "Version: " + Version + " | Commit: " + Commit + " | Built: " + BuildDate
}

func status() error {
	resp, err := getResource("/platform/status")
	if resp.StatusCode == 200 {
		fmt.Println("All systems are operational " + oddChar("😎 🚀"))
	} else {
		fmt.Println("We're having some issues right now")
	}
	return err
}

func billing() error {
	fmt.Println("Contacting billing portal... " + oddChar("💵"))
	resp, err := getResource("/platform/billing/portal")
	if resp.StatusCode >= 400 {
		panic(resp.Status)
	}
	handleError(err)
	defer resp.Body.Close()
	location := resp.Header["Location"][0]
	fmt.Println("Press any key to open the billing portal in a browser")
	openbrowser(location)
	return nil
}

func dispatchGet(resource string, args []string) {
	switch resource {
	case "status":
		status()
	default:
		log.Panic("Unknown subcommand")
	}
}

func echoData() {
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

type Context struct {
	Debug bool
}

type BillingCmd struct{}

func (r *BillingCmd) Run(ctx *Context) error {
	return billing()
}

type StatusCmd struct{}

type VersionCmd struct{}

func (r *VersionCmd) Run(ctx *Context) error {
	fmt.Println(version())
	return nil
}

func (r *StatusCmd) Run(ctx *Context) error {
	return status()
}

type EchoCmd struct {
	File string `arg optional name:"file" help:"Path to file to send to TupleStream logging platform"`
}

type TailCmd struct {
}

type DeployCmd struct {
	Target     string `arg name:"target" help:"Type of infrastructure to target. Currently the only option is 'k8s'"`
	KubeConfig string `flag name:"kubeconfig" help:"Path to kube config file. Defaults to ~/.kube/config" default:"~/.kube/config"`
	Yaml       bool   `flag name:"yaml" help:"Write proposed Kubernetes resources as yaml to stdout rather than applying them directly"`
}

func (r *DeployCmd) Run(ctx *Context) error {
	return nil
}

type LogsCmd struct {
	Echo   EchoCmd   `cmd name:"echo" help:"Directly send some log data to TupleStream from the local file system or from STDOUT"`
	Tail   TailCmd   `cmd name:"tail" help:"Tail this tenant's log stream in real-time"`
	Deploy DeployCmd `cmd name:"deploy" help:"Deploy a TupleStream logging integration"`
}

type LogoutCmd struct{}

func (r *LogoutCmd) Run(ctx *Context) error {
	removeKey()
	fmt.Println("Logged out")
	return nil
}

var CLI struct {
	Debug   bool       `help:"Print verbose log info for debugging"`
	Status  StatusCmd  `cmd help:"Get status of the TupleStream platform"`
	Billing BillingCmd `cmd help:"Open the billing portal for this tenant in a browser"`
	Logs    LogsCmd    `cmd name:"logs" help:"Interact with the TupleStream log management service for this tenant"`
	Version VersionCmd `cmd name:"version" help:"Show version information for this tuplectl build"`
	Deploy  DeployCmd  `cmd name:"deploy" help:"Deploy a TupleStream integration to your infrastructure"`
	Logout  LogoutCmd  `cmd name:"logout" help:"Log out of your TupleStream account on this machine"`
}

func main() {
	ctx := kong.Parse(&CLI)
	err := ctx.Run(&Context{Debug: CLI.Debug})
	ctx.FatalIfErrorf(err)
}
