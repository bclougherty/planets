package main

import (
	"log"
	"net/http"
	"os"
	"path"
	"runtime"

	"github.com/octoberxp/planets/controllers"

	"github.com/gorilla/mux"
	"github.com/octoberxp/glaze"
	"github.com/spf13/viper"
)

var appAddr string

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
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

	fs := http.FileServer(http.Dir("views/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Instantiate controllers
	public := controllers.NewPublicController()

	// Create routes
	router := mux.NewRouter()

	for path, handler := range glaze.GenerateRoutes(public) {
		if path == "/public/index" {
			router.HandleFunc("/", handler)
		} else {
			router.HandleFunc(path, handler)
		}
	}

	http.Handle("/", router)

	err = http.ListenAndServe(":80", router)

	if err != nil {
		log.Fatal(err)
	}
}

func errorHandler(w http.ResponseWriter, r *http.Request, status int) {
	http.Error(w, "Could not find the requested page", http.StatusInternalServerError)
}
