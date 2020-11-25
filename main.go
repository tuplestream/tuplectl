//go:generate top

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"runtime"
	"text/tabwriter"
	"time"

	"github.com/alecthomas/kong"
	"github.com/tj/go-spin"
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
	Follow   bool   `optional name:"follow" short:"f" help:"Wait for the resource to become ready after successful creation, display progress on the terminal"`
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

	if r.Follow == true {
		wait := res.Header.Get("X-Retry-After")
		createdResource := res.Header.Get("Location")
		if wait == "" || createdResource == "" {
			return nil
		}

		introString := "This can take 2-3 minutes, waiting for a bit (you can Ctrl-C this any time, run 'tuplectl get uncs' to check back " + oddChar("🙂 ") + ")"
		fmt.Println(introString)

		s := spin.New()
		tries := 0
		for tries < 10 {
			fmt.Println("")
			resource := TupleStreamResource{}
			u, err := url.Parse(createdResource)
			handleError(err)

			err = json.Unmarshal([]byte(getResourceString(u.Path)), &resource)
			handleError(err)

			if resource.State == "ready" {
				targetURL := "https://" + r.Name + ".unc.tuplestream.com"
				fmt.Println(fmt.Sprintf("%s is ready to go, press any key to open a browser.", r.Resource))
				openbrowser(targetURL)
				return nil
			}

			deadline := time.Now().Add(time.Second * time.Duration(30))

			for time.Now().Before(deadline) {
				fmt.Print("\rWaiting another 30 seconds " + s.Next())
				time.Sleep(75 * time.Millisecond)
			}

			tries++
		}

		fmt.Println(fmt.Sprintf("This is taking a little longer than usual. Check back with 'tuplectl get %s' to see when it's ready", r.Resource))
	}

	return nil
}

type GetCmd struct {
	Resource string `arg name:"resource" help:"Type of resource to retrieve"`
	ID       string `arg optional name:"id" help:"ID of specific resource to retrieve"`
	JSON     bool   `help:"return output as JSON instead of pretty-printed table"`
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

type TupleStreamResource struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	State     string `json:"state"`
}

func (r *GetCmd) Run(ctx *Context) error {
	baseURL, err := getResourceURL(r.Resource)
	if err != nil {
		return err
	}
	if r.ID == "" {
		jsonString := getResourceString(baseURL)
		deserialized := []TupleStreamResource{}
		err = json.Unmarshal([]byte(jsonString), &deserialized)
		handleError(err)
		if len(deserialized) == 0 {
			fmt.Println(fmt.Sprintf("No %s resources found for this tenant", r.Resource))
			return nil
		}
		if r.JSON {
			fmt.Print(jsonString)
		} else {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
			fmt.Fprintln(w, "NAME\tID\tSTATE\tCREATED AT\tHOSTNAME")
			for _, resource := range deserialized {
				var hostname = "-"
				if resource.State == "ready" {
					hostname = fmt.Sprintf("https://%s.unc.tuplestream.com", resource.Name)
				}
				fmt.Fprintln(w, fmt.Sprintf("%s\t%s\t%s\t%s\t%s", resource.Name, resource.ID, resource.State, resource.CreatedAt, hostname))
			}
			w.Flush()
		}
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
		if res.StatusCode == 404 {
			return errors.New("Resource not found")
		}
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
