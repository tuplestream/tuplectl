//go:generate top

package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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

func usage() {
	fmt.Println("TODO usage")
	os.Exit(1)
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

func getorListUNCs(args []string) {
	if len(args) == 0 {
		// list UNCs
		resp, err := getResource("/unc/controllers")
		handleError(err)
		defer resp.Body.Close()
		str, err := ioutil.ReadAll(resp.Body)
		handleError(err)
		fmt.Println(str)
		fmt.Println("list")
	} else {
		// get specific UNC
		id := args[0]
		resp, err := getResource("/unc/controllers/" + id)
		handleError(err)
		defer resp.Body.Close()
		str, err := ioutil.ReadAll(resp.Body)
		handleError(err)
		fmt.Println(str)
		fmt.Println("get")
	}
}

func dispatchGet(resource string, args []string) {
	switch resource {
	case "unc":
		getorListUNCs(args)
	case "status":
		status()
	default:
		log.Panic("Unknown subcommand")
	}
}

// func dispatchDelete(resource, args [string]) {
// 	switch resource {

// 	}
// }

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

var resourceBaseURLs = make(map[string]string)

func getResourceURL(resource string) (string, error) {
	baseURL := resourceBaseURLs[resource]
	if baseURL == "" {
		return "", errors.New("Unknown resource type '" + resource + "'")
	}
	return baseURL, nil
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

type CreateCmd struct {
	Resource string `arg name:"resource" help:"Type of resource to create"`
	Name     string `arg name:"name" help:"Name of the resource to create"`
}

func (r *CreateCmd) Run(ctx *Context) error {
	if r.Name == "" {
		return errors.New("Cannot create resource with no name specified")
	}
	baseURL, err := getResourceURL(r.Resource)
	if err != nil {
		return err
	}
	res, err := createResource(baseURL, "{\"name\":\""+r.Name+"\"}")
	if res.StatusCode >= 400 {
		return errors.New(res.Status)
	}
	if err != nil {
		return err
	}
	fmt.Println(r.Resource + " named '" + r.Name + "' created")
	return nil
}

type GetCmd struct {
	Resource string `arg name:"resource" help:"Type of resource to retrieve"`
	ID       string `arg optional name:"id" help:"ID of specific resource to retrieve"`
}

type EchoCmd struct {
	File string `arg optional name:"file" help:"Path to file to send to TupleStream logging platform"`
}

type TailCmd struct {
}

type LogsCmd struct {
	Echo EchoCmd `cmd name:"echo" help:"Directly send some log data to TupleStream from the local file system or from STDOUT"`
	Tail TailCmd `cmd name:"tail" help:"Tail this tenant's log stream in real-time"`
}

func (r *GetCmd) Run(ctx *Context) error {
	baseURL, err := getResourceURL(r.Resource)
	if err != nil {
		return err
	}
	if r.ID == "" {
		fmt.Println(getResourceString(baseURL))
	} else {
		fmt.Println(getResourceString(baseURL + "/" + r.ID))
	}
	return nil
}

type DeleteCmd struct {
	Resource string `arg name:"resource" help:"Type of resource to delete"`
	ID       string `arg name:"id" help:"ID of the resource to delete"`
}

func (r *DeleteCmd) Run(ctx *Context) error {
	baseURL, err := getResourceURL(r.Resource)
	if err != nil {
		return err
	}

	res, err := deleteResource(baseURL + "/" + r.ID)
	if err != nil {
		return err
	}
	if res.StatusCode >= 400 {
		return errors.New("Request error: " + res.Status)
	}
	fmt.Println("Successfully deleted " + r.Resource)
	return nil
}

var CLI struct {
	Debug   bool       `help:"Print verbose log info for debugging"`
	Status  StatusCmd  `cmd help:"Get status of the TupleStream platform"`
	Billing BillingCmd `cmd help:"Open the billing portal for this tenant in a browser"`
	Create  CreateCmd  `cmd name:"create" help:"Create a resource for this tenant"`
	Delete  DeleteCmd  `cmd name:"delete" help:"Delete a resource belonging to this tenant"`
	Get     GetCmd     `cmd name:"get" help:"Get or list a resource belonging to this tenant"`
	Logs    LogsCmd    `cmd name:"logs" help:"Interact with the TupleStream log management service for this tenant"`
	Version VersionCmd `cmd name:"version" help:"Show version information for this tuplectl build"`
}

func main() {
	resourceBaseURLs["unc"] = "/unc/controllers"
	resourceBaseURLs["uncs"] = "/unc/controllers"

	ctx := kong.Parse(&CLI)
	err := ctx.Run(&Context{Debug: CLI.Debug})
	ctx.FatalIfErrorf(err)
}
