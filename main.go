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

var (
	validate   *validator.Validate
	httpClient = &http.Client{
		Timeout: time.Second * 10,
	}
	latLngPattern = regexp.MustCompile(`^[-+]?([1-8]?\d(\.\d+)?|90(\.0+)?),[-+]?(180(\.0+)?|((1[0-7]\d)|([1-9]?\d))(\.\d+)?)$`)
	osrmApiUrl    = "http://router.project-osrm.org/route/v1/driving/%s;%s?overview=false"
)

type QueryParams struct {
	Src string   `form:"src" binding:"required" validate:"latlng"`
	Dst []string `form:"dst" binding:"required" validate:"latlng"`
}

type OsrmApiRouteData struct {
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

type GetRoutesResp struct {
	Source string  `json:"source"`
	Routes []Route `json:"routes"`
}

type ErrResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func setupRouter() *gin.Engine {
	r := gin.Default()

	validate = validator.New()
	validate.RegisterValidation("latlng", validateLatLng)

	r.GET("/routes", getRoutes)

	return r
}

func main() {
	r := setupRouter()
	r.Run(":8080")
}

func getRoutes(c *gin.Context) {

	var query QueryParams

	err := c.ShouldBindQuery(&query)
	if err == nil {
		err = validate.Struct(query)
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, ErrResp{
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

	var resp = GetRoutesResp{
		Source: query.Src,
		Routes: routes,
	}

	resp.sortRoutesByDurationAsc()

	c.JSON(http.StatusOK, resp)

}

func (o *GetRoutesResp) sortRoutesByDurationAsc() {
	sort.Slice(o.Routes, func(i, j int) bool {
		// Sort by duration if distance is equal
		if o.Routes[i].Duration == o.Routes[j].Duration {
			return o.Routes[i].Distance < o.Routes[j].Distance
		}

		// Sort by duration
		return o.Routes[i].Duration < o.Routes[j].Duration
	})
}

func fetchRouteData(src string, dst string, routeCh chan Route) {
	url := fmt.Sprintf(osrmApiUrl, src, dst)
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

	var data OsrmApiRouteData
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
