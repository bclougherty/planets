package controllers

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/octoberxp/glaze"
	"github.com/octoberxp/swplanetgen"
	"github.com/spf13/viper"
)

// Public Controller
type Public struct {
	*glaze.Controller
}

// NewPublicController creates and returns the new public controller
func NewPublicController() *Public {
	controller, err := glaze.NewController(viper.GetString("FullViewPath"), "public")
	if err != nil {
		panic(err)
	}

	return &Public{Controller: controller}
}

type indexData struct {
	Categories  map[string]*swplanetgen.Result
	HoursPerDay int
	DaysPerYear int
	Population  int
	Error       error
}

// Index page
func (controller *Public) Index(w http.ResponseWriter, r *http.Request) {
	rand.Seed(time.Now().UTC().UnixNano())

	planet, err := swplanetgen.GeneratePlanet(viper.GetString("DatabaseConnectionString"))
	if err != nil {
		viewData := &indexData{
			Error: err,
		}

		controller.RenderTemplate(w, "error", viewData)
	} else {
		viewData := &indexData{
			Categories: map[string]*swplanetgen.Result{
				"Function":    planet.Function,
				"Government":  planet.Government,
				"Type":        planet.Type,
				"Terrain":     planet.Terrain,
				"Temperature": planet.Temperature,
				"Gravity":     planet.Gravity,
				"Atmosphere":  planet.Atmosphere,
				"Hydrosphere": planet.Hydrosphere,
				"Starport":    planet.Starport,
				"Tech Level":  planet.TechLevel,
			},
			HoursPerDay: planet.HoursPerDay,
			DaysPerYear: planet.DaysPerYear,
			Population:  planet.Population,
			Error:       err,
		}

		controller.RenderTemplate(w, "index", viewData)
	}
}
