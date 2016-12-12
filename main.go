package main

import (
	"archive/tar"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"text/template"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

var (
	version, repo, bin               string
	fetch, invoke, build, get, tools bool
)

const (
	fetchDf = `FROM golang:{{.Version}}
LABEL builder=dogo
RUN go get -u {{.Repo}} && \
    mv /go/bin/{{.Bin}} /bin/{{.Bin}}
ENTRYPOINT ["{{.Bin}}"]
`

	invokeSh = `{{.Bin}}() {
  docker run -it --name dogo-runtime -v "$(pwd)":/pwd -w /pwd dogo/{{.Bin}}:latest $@
  docker rm -vf dogo-runtime >/dev/null
}`
	invokeToolsSh = `go() {
  docker run -it --name dogo-runtime -v "$(pwd)":/pwd -w /pwd golang:{{.Version}} go $@
  docker rm -vf dogo-runtime >/dev/null
}`
)

type wrapped struct {
	Version string
	Bin     string
	Repo    string
}

func init() {
	vpt := flag.String(`r`, `1.7.4`, `The golang release version to base on.`)
	fpt := flag.Bool(`df`, false, `Display the fetching Dockerfile.`)
	//ipt := flag.Bool(`sh`, false, `Display the invoking shell function.`)

	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println(`Expected at least one argument.`)
		flag.PrintDefaults()
		os.Exit(1)
	} else if flag.NArg() == 1 {
		if flag.Arg(0) == `tools` {
			tools = true
		} else {
			repo = flag.Arg(0)
		}
	} else if flag.NArg() == 2 {
		if flag.Arg(0) == `build` {
			repo = flag.Arg(1)
			build = true
		} else if flag.Arg(0) == `get` {
			repo = flag.Arg(1)
			get = true
		} else {
			panic(`Unknown command.`)
		}
	}

	version = *vpt
	fetch = *fpt
	//invoke = *ipt

	var p = regexp.MustCompile(`^([^/]+/)*([^/]+)/?$`)
	ps := p.FindStringSubmatch(repo)
	if !tools {
		bin = ps[len(ps)-1]
	}
}

func main() {
	var in = wrapped{
		Version: version,
		Bin:     bin,
		Repo:    repo,
	}

	if fetch {
		doDockerfile(in)
	} else if invoke {
		doShell(in)
	} else if build {
		doBuild(in)
	} else if get {
		doBuild(in)
		doShell(in)
	} else if tools {
		doToolsShell(in)
	}
}

func doDockerfile(data wrapped) {
	ft := template.Must(template.New("fetchDf").Parse(fetchDf))
	if err := ft.Execute(os.Stdout, data); err != nil {
		fmt.Println(err)
	}
}

func doShell(data wrapped) {
	it := template.Must(template.New("invokeSh").Parse(invokeSh))
	if err := it.Execute(os.Stdout, data); err != nil {
		fmt.Println(err)
	}
}

func doToolsShell(data wrapped) {
	it := template.Must(template.New("invokeToolsSh").Parse(invokeToolsSh))
	if err := it.Execute(os.Stdout, data); err != nil {
		fmt.Println(err)
	}
}

func doBuild(data wrapped) {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	ft := template.Must(template.New("fetchDf").Parse(fetchDf))

	// Generate a dockerfile
	var df bytes.Buffer
	if err := ft.Execute(&df, data); err != nil {
		panic(err)
	}

	// Prepare a build context (tar)
	var raw bytes.Buffer
	tw := tar.NewWriter(&raw)
	hdr := &tar.Header{
		Name: `Dockerfile`,
		Mode: 0600,
		Size: int64(df.Len()),
	}
	if err = tw.WriteHeader(hdr); err != nil {
		panic(err)
	}
	if _, err = tw.Write(df.Bytes()); err != nil {
		panic(err)
	}
	if err = tw.Close(); err != nil {
		panic(err)
	}
	tar := bytes.NewReader(raw.Bytes())

	// Perform the build
	br, err := cli.ImageBuild(context.Background(),
		tar,
		types.ImageBuildOptions{
			Tags:        []string{`dogo/` + bin},
			Remove:      true,
			ForceRemove: true,
		})
	if err != nil {
		panic(err)
	}
	_, err = ioutil.ReadAll(br.Body)
	defer br.Body.Close()
}
