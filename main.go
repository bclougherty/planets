package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/octoberxp/planets/controllers"

	"github.com/gorilla/mux"
	"github.com/octoberxp/glaze"
	"github.com/spf13/viper"
)

var port int

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.IntVar(&port, "port", 80, "What port to listen on")
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	// Create configuration
	viper.Set("AppRoot", os.ExpandEnv("."))

	viper.SetConfigName("config")
	viper.AddConfigPath(os.ExpandEnv("./config/"))
	err := viper.ReadInConfig()

	if err != nil {
		log.Fatalf("Error reading in configuration: %s", err)
		return
	}

	viper.Set("FullViewPath", path.Join(viper.GetString("AppRoot"), viper.GetString("ViewPath")))

	funcMap := template.FuncMap{
		"safeHtml": glaze.SafeHTML,
	}

	// Instantiate controllers
	public := controllers.NewPublicController(funcMap)

	// Create routes
	router := mux.NewRouter()

	for path, handler := range glaze.GenerateRoutes(public) {
		if path == "/public/index" {
			router.HandleFunc("/", handler)
		} else {
			router.HandleFunc(path, handler)
		}
	}

	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("views/static"))))

	http.Handle("/", router)

	flag.Parse()

	err = http.ListenAndServe(fmt.Sprintf(":%d", port), router)

	if err != nil {
		log.Fatal(err)
	}
}

func errorHandler(w http.ResponseWriter, r *http.Request, status int) {
	http.Error(w, "Could not find the requested page", http.StatusInternalServerError)
}
