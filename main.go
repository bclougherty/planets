package main

import (
	"html/template"
	"log"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/codegangsta/cli"
	"github.com/octoberxp/planets/controllers"

	"github.com/octoberxp/glaze"
)

var port int
var verbose, graceful bool

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().UTC().UnixNano())
}

func main() {
	app := cli.NewApp()
	app.Name = "Planet Generator"
	app.Usage = "Generates random planets from a galaxy far, far, away"

	app.HideHelp = true

	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:        "port",
			Value:       80,
			Usage:       "The port this server will run on",
			Destination: &port,
		},
		cli.BoolFlag{
			Name:        "verbose",
			Usage:       "Enables verbose logging",
			Destination: &verbose,
		},
		cli.BoolFlag{
			Name:        "graceful",
			Usage:       "Internal use only",
			Destination: &graceful,
		},
	}

	app.Action = startServer

	app.Run(os.Args)
}

func startServer(c *cli.Context) {
	var glazeApp glaze.Application

	err := glazeApp.Initialize(os.Args[0], c.Int("port"), c.Bool("verbose"), c.Bool("graceful"))
	if nil != err {
		panic(err)
	}

	mainLog, err := glazeApp.CreateLog("planets.log")
	if err != nil {
		panic(err)
	}

	log.SetOutput(mainLog)

	// Instantiate controllers
	templateFuncs := template.FuncMap{
		"SafeHtml": glaze.SafeHTML,
	}

	public, err := controllers.NewPublicController(templateFuncs)
	if nil != err {
		panic(err)
	}

	allControllers := []*glaze.Controller{public}

	// Start glaze
	err = glazeApp.ServeWithAutoRouting(allControllers)
	if nil != err {
		panic(err)
	}
}
