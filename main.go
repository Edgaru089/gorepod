package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	_ "embed"
)

type Repo struct {
	ImportPath string // like example.com/package/code
	ShortPath  string // like package/code

	RepoType string // like git or svn
	RepoPath string // like https://git.example.com/package/code.git or git@git.example.com:package/code.git
}

type Config struct {
	GitServerPrefix string // Like https://github.com or https://git.example.com (without ending slash)

	// go-source tag strings. Example:
	//     GoSourceFile: https://git.example.com/{/path}/src/branch/master{/dir}
	//     GoSourceLine: https://git.example.com/{/path}/src/branch/master{/dir}/{file}#L{line}
	GoSourceFile, GoSourceLine string

	Repos map[string]*Repo // mapped by ShortPath, like package/code
}

//go:embed template.html
var httpTemplate string

var (
	ConfigFile, TemplateFile string
	ListenAddress            string
)

func init() {
	flag.StringVar(&ConfigFile, "config", "config.json", "Config file")
	flag.StringVar(&TemplateFile, "template", "template.html", "template HTML file")
	flag.StringVar(&ListenAddress, "listen", "0.0.0.0:32148", "listen address")
}

type Server struct {
	config   Config
	template *template.Template
}

func (s *Server) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	log.Print("request: ", req.URL.Path)
	id := req.URL.Path[1:]
	if repo, ok := s.config.Repos[id]; ok {
		err := s.template.Execute(resp, repo)
		if err != nil {
			resp.WriteHeader(500)
			resp.Write([]byte("Internal Server Error\n\n"))
			resp.Write([]byte(fmt.Sprintf("Error: Executing Template: %s, id=%s\n\nRepo=%v", err.Error(), id, repo)))
		}
	} else {
		resp.WriteHeader(404)
	}
}

func main() {
	var c Config

	flag.Parse()

	file, err := os.Open(ConfigFile)
	if err != nil {
		fmt.Fprint(os.Stderr, "cannot open config file: ", err)
		os.Exit(1)
	}

	dec := json.NewDecoder(file)
	err = dec.Decode(&c)
	if err != nil {
		fmt.Fprint(os.Stderr, "error parsing config file:", err)
		os.Exit(1)
	}

	tempbyte, err := os.ReadFile(TemplateFile)
	if err != nil {
		fmt.Fprint(os.Stderr, "error reading template file:", err)
		os.Exit(1)
	}

	s := &Server{config: c, template: template.New("")}
	s.template.Funcs(template.FuncMap{
		"config": func() Config {
			return c
		},
	})
	_, err = s.template.Parse(string(tempbyte))
	if err != nil {
		fmt.Fprint(os.Stderr, "error parsing template file:", err)
		os.Exit(1)
	}

	log.Print("listening on ", ListenAddress)
	http.ListenAndServe(ListenAddress, s)

}