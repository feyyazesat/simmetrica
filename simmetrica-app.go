package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	simmetrica "github.com/feyyazesat/simmetrica/simmlib"
	httprouter "github.com/julienschmidt/httprouter"
	yamlLib "gopkg.in/yaml.v2"
)

type (
	Yaml struct {
		Graphs []map[string]interface{}
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
	/*
		tmpl, err := template.New("Index").ParseFiles(TEMPLATE_FOLDER + "/index.html")
		Check(err)
		/*
			data := struct {
				Title string
				Body  string
			}{
				"About page",
				"Body info",
			}

		err = tmpl.Execute(w, nil)
		Check(err)
	*/
	fmt.Fprint(w, string(*tmpl))
}

func Push(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	increment, _ := strconv.ParseUint(r.FormValue("increment"), 10, 0)
	now, _ := strconv.ParseUint(r.FormValue("now"), 10, 0)

	if !(increment > 0) {
		increment = simmetrica.DEFAULT_INCREMENT
	}
	pushreply, err := simmetrica.Push(params.ByName("event"), increment, now)
	Check(err)

	//debugging.
	_ = pushreply

	fmt.Fprint(w, "ok")
}

func Query(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	fmt.Fprint(w, params.ByName("event"), params.ByName("start"), params.ByName("end"))
}

func Graph(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, yaml)
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
		&simmetrica.RedisArgs.RedisHost,
		"redis_host",
		simmetrica.DEFAULT_REDIS_HOST,
		"Connect to redis on the specified host")

	flag.StringVar(
		&simmetrica.RedisArgs.RedisPort,
		"redis_port",
		simmetrica.DEFAULT_REDIS_PORT,
		"Connect to redis on the specified port")

	flag.StringVar(
		&simmetrica.RedisArgs.RedisDb,
		"redis_db",
		simmetrica.DEFAULT_REDIS_DB,
		"Connect to the specified db in redis")

	flag.StringVar(
		&simmetrica.RedisArgs.RedisPassword,
		"redis_password",
		simmetrica.DEFAULT_REDIS_PASSWORD,
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
	conn := simmetrica.Initialize()
	defer simmetrica.Uninitialize()(*conn)

	router = CreateRoutes(httprouter.New())
	fmt.Println("Server Started at 127.0.0.1:8080 Time : ", simmetrica.GetCurrentTimeStamp())
	log.Fatal(http.ListenAndServe(":8080", router))

}
