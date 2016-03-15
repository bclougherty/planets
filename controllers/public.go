package controllers

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/octoberxp/glaze"
	"github.com/octoberxp/swplanetgen"
	"github.com/spf13/viper"
)

// Public Controller
type Public struct {
	*glaze.Controller
}

// NewPublicController creates and returns the new public controller
func NewPublicController(funcMap template.FuncMap) (*Public, error) {
	controller, err := glaze.NewController(viper.GetString("FullViewPath"), "public", funcMap)
	if err != nil {
		return nil, err
	}

	return &Public{Controller: controller}, nil
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
	requestedParams := make(map[int]string)
	possibleParams := []int{
		swplanetgen.CategoryFunction,
		swplanetgen.CategoryGovernment,
		swplanetgen.CategoryType,
		swplanetgen.CategoryTerrain,
		swplanetgen.CategoryTemperature,
		swplanetgen.CategoryGravity,
		swplanetgen.CategoryAtmosphere,
		swplanetgen.CategoryHydrosphere,
		swplanetgen.CategoryStarport,
		swplanetgen.CategoryTechlevel,
	}

	for _, category := range possibleParams {
		if val := r.URL.Query().Get(fmt.Sprintf("%d", category)); len(val) > 0 {
			requestedParams[category] = val
		}
	}

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

		err = controller.RenderTemplate(w, "index", viewData)
		if err != nil {
			fmt.Println(err)
		}
	}
}
