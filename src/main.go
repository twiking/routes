package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"sync"
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
	Code    string `json:"code"`
	Message string `json:"message"`
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
	r.Run()
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
			Message: validationErrMsg(err),
		})
		return
	}

	routes := make([]Route, 0)

	var wg sync.WaitGroup
	for _, dst := range query.Dst {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			route, err := getRouteData(query.Src, d)
			if err != nil {
				// Here we could save errors to a []Error and handle them depending on requirements.
				// For now, no individual errors will block the output.
			} else {
				routes = append(routes, route)
			}
		}(dst)
	}

	wg.Wait()

	var resp = GetRoutesResp{
		Source: query.Src,
		Routes: routes,
	}

	resp.sortRoutesByDurationAsc()

	c.JSON(http.StatusOK, resp)
}

func getRouteData(src string, dst string) (Route, error) {
	url := fmt.Sprintf(osrmApiUrl, src, dst)

	resp, body, err := makeRequestWith429Retries(url)
	if err != nil {
		return Route{}, err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		return Route{}, fmt.Errorf("response code: %d", resp.StatusCode)
	}

	var data OsrmApiRouteData
	err = json.Unmarshal(body, &data)
	if err != nil {
		return Route{}, err
	}

	if data.Code != "Ok" {
		return Route{}, fmt.Errorf("response code: %d. message: %s", resp.StatusCode, data.Message)
	}

	route := Route{
		Destination: dst,
		Duration:    data.Routes[0].Duration,
		Distance:    data.Routes[0].Distance,
	}

	return route, nil
}

func makeRequestWith429Retries(url string) (*http.Response, []byte, error) {
	var (
		body []byte
		err  error
		resp *http.Response
	)
	attempts := 20
	backoffTime := 1 * time.Second

	for i := 0; i < attempts; i++ {
		resp, err = httpClient.Get(url)
		if err != nil {
			return nil, nil, err
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			time.Sleep(backoffTime)
			continue
		}

		defer resp.Body.Close()
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, err
		}
		break
	}

	return resp, body, nil
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

// latLng should have the pattern 13.388860,52.517037
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

func validationErrMsg(err error) string {
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
