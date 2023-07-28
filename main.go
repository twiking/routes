package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type QueryParams struct {
	Src string   `form:"src" binding:"required" validate:"latlng"`
	Dst []string `form:"dst" binding:"required" validate:"latlng"`
}

type ExternalRouteData struct {
	Routes []struct {
		Duration float64 `json:"duration"`
		Distance float64 `json:"distance"`
	} `json:"routes"`
	Code string `json:"code"`
}

type Route struct {
	Destination string  `json:"destination"`
	Duration    float64 `json:"duration"`
	Distance    float64 `json:"distance"`
}

type Output struct {
	Source string  `json:"source"`
	Routes []Route `json:"routes"`
}

type ErrorOutput struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

var (
	validate   *validator.Validate
	httpClient = &http.Client{
		Timeout: time.Second * 10,
	}
	latLngPattern = regexp.MustCompile(`^[-+]?([1-8]?\d(\.\d+)?|90(\.0+)?),[-+]?(180(\.0+)?|((1[0-7]\d)|([1-9]?\d))(\.\d+)?)$`)
)

func main() {
	router := gin.Default()

	validate = validator.New()
	validate.RegisterValidation("latlng", validateLatLng)

	router.GET("/routes", getRoutes)
	router.Run(":8080")
}

func getRoutes(c *gin.Context) {

	var query QueryParams

	err := c.ShouldBindQuery(&query)
	if err == nil {
		err = validate.Struct(query)
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorOutput{
			Code:    http.StatusBadRequest,
			Message: customErrorMessage(err),
		})
		return
	}

	routeCh := make(chan Route)
	routes := make([]Route, 0)

	for _, dst := range query.Dst {
		go fetchRouteData(query.Src, dst, routeCh)
	}

	for i := 0; i < len(query.Dst); i++ {
		route := <-routeCh
		if route != (Route{}) {
			routes = append(routes, route)
		}
	}

	sortRoutes(routes)

	c.JSON(http.StatusOK, Output{
		Source: query.Src,
		Routes: routes,
	})

}

func sortRoutes(routes []Route) {
	sort.Slice(routes, func(i, j int) bool {
		// Sort by duration if distance is equal
		if routes[i].Duration == routes[j].Duration {
			return routes[i].Distance < routes[j].Distance
		}

		// Sort by duration
		return routes[i].Duration < routes[j].Duration
	})
}

func fetchRouteData(src string, dst string, routeCh chan Route) {
	url := fmt.Sprintf("http://router.project-osrm.org/route/v1/driving/%s;%s?overview=false", src, dst)
	resp, err := httpClient.Get(url)

	if (err != nil) || (resp.StatusCode != http.StatusOK) {
		routeCh <- Route{}
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		routeCh <- Route{}
		return
	}

	var data ExternalRouteData
	err = json.Unmarshal(body, &data)
	if (err != nil) || (data.Code != "Ok") {
		routeCh <- Route{}
		return
	}

	route := Route{
		Destination: dst,
		Duration:    data.Routes[0].Duration,
		Distance:    data.Routes[0].Distance,
	}

	routeCh <- route
}

func validateLatLng(fl validator.FieldLevel) bool {
	switch v := fl.Field().Interface().(type) {
	case string:
		return latLngPattern.MatchString(v)
	case []string:
		for _, str := range v {
			match := latLngPattern.MatchString(str)
			if !match {
				return false
			}
		}
		return true
	default:
		// Unknown type
		return false
	}
}

func customErrorMessage(err error) string {
	errs := err.(validator.ValidationErrors)
	for _, e := range errs {
		switch e.Tag() {
		case "required":
			return fmt.Sprintf("%s is a required field", e.Field())
		case "latlng":
			return fmt.Sprintf("%s is not a valid latitude and longitude", e.Field())
		default:
			return fmt.Sprintf("%s is not valid", e.Field())
		}
	}

	return "Unknown error"
}
