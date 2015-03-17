package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/feyyazesat/simmetrica/simmlib"
	"github.com/julienschmidt/httprouter"
	yamlLib "gopkg.in/yaml.v2"
)

type (
	Yaml struct {
		Graphs []struct {
			Title         string
			Timespan      string
			Colorscheme   string
			Type          string
			Interpolation string
			Resolution    string
			Size          string
			Offset        string
			Events        []struct {
				Name  string
				Title string
			}
		}
	}
	graphEvents struct {
		Name  string                   `json:"name"`
		Title string                   `json:"title"`
		Data  []simmlib.TstampValTuple `json:"data"`
	}

	graphResult struct {
		Title         string        `json:"title"`
		Colorscheme   string        `json:"colorscheme"`
		Type          string        `json:"type"`
		Interpolation string        `json:"interpolation"`
		Resolution    string        `json:"resolution"`
		Size          string        `json:"size"`
		Offset        string        `json:"offset"`
		Identifier    string        `json:"identifier"`
		Events        []graphEvents `json:"events"`
	}

	justFilesFilesystem struct {
		fs http.FileSystem
	}
	neuteredReaddirFile struct {
		http.File
	}

	httprouterReturn func(w http.ResponseWriter, r *http.Request, param httprouter.Params)
)

const (
	//	STATIC_FOLDER = "/opt/simmetrica/static"
	//	TEMPLATE_FOLDER = "/opt/simmetrica/templates"
	//	DEFAULT_CONFIG_FILE = "/opt/simmetrica/config/config.yml"

	STATIC_FOLDER       = "/var/golang/src/github.com/feyyazesat/simmetrica/static"
	TEMPLATE_FOLDER     = "/var/golang/src/github.com/feyyazesat/simmetrica/templates"
	DEFAULT_CONFIG_FILE = "/var/golang/src/github.com/feyyazesat/simmetrica/config/config.yml"
)

var (
	err error

	yaml Yaml

	args struct {
		debug      bool
		configPath string
	}

	router *httprouter.Router
)

func Check(err error) {
	if err != nil {
		panic(err)
		os.Exit(1)
	}
}

func LogWrapper(fnHandler httprouterReturn, name string) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, param httprouter.Params) {

		start := time.Now()

		fnHandler(w, r, param)

		log.Printf(
			"%s\t%s\t%s\t%s",
			r.Method,
			r.RequestURI,
			name,
			time.Since(start),
		)
	}
}

func (yaml *Yaml) UnmarshalYAML(content *[]byte) error {

	return yamlLib.Unmarshal(*content, yaml)

}

func ReadFile(path string) (*[]byte, error) {
	content, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, err
	}

	return &content, nil
}

func (fs justFilesFilesystem) Open(name string) (http.File, error) {
	f, err := fs.fs.Open(name)
	if err != nil {
		return nil, err
	}

	return neuteredReaddirFile{f}, nil
}

func (f neuteredReaddirFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	tmpl, err := ReadFile(TEMPLATE_FOLDER + "/index.html")
	Check(err)
	fmt.Fprint(w, string(*tmpl))
}

func Push(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	increment, _ := strconv.ParseUint(r.FormValue("increment"), 10, 0)
	now, _ := strconv.ParseUint(r.FormValue("now"), 10, 0)

	if !(increment > 0) {
		increment = simmlib.DEFAULT_INCREMENT
	}
	_, err := simmlib.Push(params.ByName("event"), increment, now)
	Check(err)

	fmt.Fprint(w, "ok")
}

func Query(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	fmt.Fprint(w, params.ByName("event"), params.ByName("start"), params.ByName("end"))
}

func Graph(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	now := simmlib.GetCurrentTimeStamp()
	var timespanAsSec, start, end uint64
	var series []simmlib.TstampValTuple
	var events []graphEvents
	var result graphResult
	var results []graphResult

	for _, sections := range yaml.Graphs {
		if sections.Timespan == "" {
			sections.Timespan = "1 day"
		}
		timespanAsSec = simmlib.GetSecFromRelativeTime(sections.Timespan)
		events = events[:0]
		for _, event := range sections.Events {
			start = now - timespanAsSec
			end = now + simmlib.GetResolution(sections.Resolution)
			series = simmlib.Query(event.Name, start, end, simmlib.GetResolutionKey(sections.Resolution))
			events = append(events, graphEvents{Name: event.Name, Title: event.Title, Data: series})
		}
		result = graphResult{
			Title:         sections.Title,
			Colorscheme:   sections.Colorscheme,
			Type:          sections.Type,
			Interpolation: sections.Interpolation,
			Resolution:    sections.Resolution,
			Size:          sections.Size,
			Offset:        sections.Offset,
			Events:        events,
		}
		results = append(results, result)
	}
	response, err := json.MarshalIndent(results, "", "  ")
	Check(err)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(response))
}

func CreateRoutes(router *httprouter.Router) *httprouter.Router {
	router.GET("/", LogWrapper(Index, "Index"))
	router.GET("/push/:event", LogWrapper(Push, "Push"))
	router.GET("/query/:event/:start/:end", LogWrapper(Query, "Query"))
	router.GET("/graph", LogWrapper(Graph, "Graph"))

	fs := justFilesFilesystem{http.Dir(STATIC_FOLDER)}
	router.Handler("GET", "/static/*filepath", http.StripPrefix("/static", http.FileServer(fs)))

	return router
}

func init() {
	flag.BoolVar(
		&args.debug,
		"debug",
		false,
		"Run the app in debug mode")

	flag.StringVar(
		&args.configPath,
		"config",
		DEFAULT_CONFIG_FILE,
		fmt.Sprintf("Run with the specified config file (default: %s)", DEFAULT_CONFIG_FILE))

	flag.StringVar(
		&simmlib.RedisArgs.RedisHost,
		"redis_host",
		simmlib.DEFAULT_REDIS_HOST,
		"Connect to redis on the specified host")

	flag.StringVar(
		&simmlib.RedisArgs.RedisPort,
		"redis_port",
		simmlib.DEFAULT_REDIS_PORT,
		"Connect to redis on the specified port")

	flag.StringVar(
		&simmlib.RedisArgs.RedisDb,
		"redis_db",
		simmlib.DEFAULT_REDIS_DB,
		"Connect to the specified db in redis")

	flag.StringVar(
		&simmlib.RedisArgs.RedisPassword,
		"redis_password",
		simmlib.DEFAULT_REDIS_PASSWORD,
		"Authorization password of redis")

	flag.Parse()
}

func main() {
	{
		//scope to unset configContent
		var configContent *[]byte
		configContent, err = ReadFile(args.configPath)
		Check(err)

		err = yaml.UnmarshalYAML(configContent)
		Check(err)
	}
	simmlib.Initialize()
	defer simmlib.Uninitialize()

	router = CreateRoutes(httprouter.New())
	fmt.Println("Server Started at 127.0.0.1:8080 Time : ", simmlib.GetCurrentTimeStamp())
	log.Fatal(http.ListenAndServe(":8080", router))

}
