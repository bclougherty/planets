package core

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gorilla/mux"
	"github.com/octoberxp/glaze"
	"github.com/octoberxp/go-utils/stringutils"
)

type ControllerMapper struct {
	controllers       map[string]interface{}
	controllerActions map[string][]string

	ErrorHandler func(w http.ResponseWriter, r *http.Request, status int)
}

func NewControllerMapper(controllerMap map[string]interface{}) *ControllerMapper {
	mapper := &ControllerMapper{}

	mapper.buildMap(controllerMap)

	return mapper
}

func (mapper *ControllerMapper) buildMap(controllerMap map[string]interface{}) {
	if mapper.controllerActions != nil {
		return
	}

	mapper.controllerActions = make(map[string][]string)
	mapper.controllers = controllerMap

	// create a Glaze controller and get a list of its methods
	// so that we can exclude them from the list of handle-able methods
	glazeController := &glaze.Controller{}

	glazeMethods := methodsOfStruct(glazeController, []string{})

	// fmt.Printf("GLAZE METHODS: %s\n", glazeMethods)

	for controllerName, controller := range controllerMap {
		mapper.controllerActions[controllerName] = methodsOfStruct(controller, glazeMethods)
	}
}

func (mapper *ControllerMapper) HandleIfPossible(w http.ResponseWriter, r *http.Request) {
	requestVars := mux.Vars(r)
	controller := requestVars["controller"]
	action := stringutils.SpinalToCamel(requestVars["action"])

	if _, exists := mapper.controllerActions[controller]; !exists {
		// fmt.Print("could not find controller %s\n", controller)

		if mapper.ErrorHandler != nil {
			mapper.ErrorHandler(w, r, http.StatusNotFound)
		} else {
			errorHandler(w, r, http.StatusNotFound)
		}
	}

	for _, a := range mapper.controllerActions[controller] {
		if a == action {
			inputs := make([]reflect.Value, 2)
			inputs[0] = reflect.ValueOf(w)
			inputs[1] = reflect.ValueOf(r)
			reflect.ValueOf(mapper.controllers[controller]).MethodByName(action).Call(inputs)

			// fmt.Printf("calling action %s.%s\n", controller, action)

			return
		}
	}

	// fmt.Printf("could not find action %s of controller %s\n", action, controller)

	if mapper.ErrorHandler != nil {
		// fmt.Print("error handler is not nil\n")
		mapper.ErrorHandler(w, r, http.StatusNotFound)
	} else {
		errorHandler(w, r, http.StatusNotFound)
	}
}

func (mapper *ControllerMapper) ActionMap() string {
	output := make([]string, 0)

	for controllerName, methods := range mapper.controllerActions {
		for i := 0; i < len(methods); i++ {
			output = append(output, fmt.Sprintf("\"/%s/%s\", %s.%s", controllerName, stringutils.CamelToSpinal(methods[i]), controllerName, methods[i]))
		}
	}

	return strings.Join(output, "\n")
}

func methodsOfStruct(theStruct interface{}, exclude []string) []string {
	structType := reflect.TypeOf(theStruct)
	numberOfMethods := structType.NumMethod()

	var actions = make([]string, 0)

	for i := 0; i < numberOfMethods; i++ {
		method := structType.Method(i)

		if method.Name != "" && !contains(exclude, method.Name) {
			actions = append(actions, method.Name)
		}
	}

	return actions
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func errorHandler(w http.ResponseWriter, r *http.Request, status int) {
	http.Error(w, "Error 404 - Could not find the requested page", http.StatusInternalServerError)
}
