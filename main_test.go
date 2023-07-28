package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRoutesReturns400WhenNoParams(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/routes", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetRoutesReturns400WhenNoSrcParam(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/routes?dst=13.397634,52.529407", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetRoutesReturns400WhenNoDstParam(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/routes?src=13.388860,52.517037", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetRoutesReturns400WhenLatLongIsInvalid(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/routes?src=13.388860,52.517037&dst=13.428555,52.523219&dst=invalid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetRoutesReturns200(t *testing.T) {
	osrmApiPath := "/route/v1/driving/%s;%s"
	src := "13.388860,52.517037"
	dst1 := "13.397634,52.529407"
	dst2 := "13.428555,52.523219"
	dst3 := "13.428555,48.523219"

	mockOsrmApi := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p1 := fmt.Sprintf(osrmApiPath, src, dst1)
		if r.URL.Path == p1 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"code":"Ok", "routes": [{"duration":2490.1,"distance":3286.3}]}`))
			return
		}

		p2 := fmt.Sprintf(osrmApiPath, src, dst2)
		if r.URL.Path == p2 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"code":"Ok", "routes": [{"duration":260.1,"distance":1886.3}]}`))
			return
		}

		p3 := fmt.Sprintf(osrmApiPath, src, dst3)
		if r.URL.Path == p3 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"code":"InvalidQuery"}`))
			return
		}

	}))
	defer mockOsrmApi.Close()

	router := setupRouter()
	osrmApiUrl = mockOsrmApi.URL + osrmApiPath

	w := httptest.NewRecorder()
	url := fmt.Sprintf("/routes?src=%s&dst=%s&dst=%s&dst=%s", src, dst1, dst2, dst3)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	expectedResp := `{"source":"13.388860,52.517037","routes":[{"destination":"13.428555,52.523219","duration":260.1,"distance":1886.3},{"destination":"13.397634,52.529407","duration":2490.1,"distance":3286.3}]}`
	assert.Equal(t, expectedResp, w.Body.String())
}

func TestSortRoutesByDurationAsc(t *testing.T) {
	routes := []Route{
		{"13.397634,52.529407", 500, 100},
		{"13.397634,52.529407", 200, 300},
		{"13.397634,52.529407", 200, 100},
		{"13.397634,52.529407", 100, 10},
		{"13.397634,52.529407", 200, 50},
		{"13.397634,52.529407", 200, 100},
		{"13.397634,52.529407", 100, 100},
	}

	expectedRoutes := []Route{
		{"13.397634,52.529407", 100, 10},
		{"13.397634,52.529407", 100, 100},
		{"13.397634,52.529407", 200, 50},
		{"13.397634,52.529407", 200, 100},
		{"13.397634,52.529407", 200, 100},
		{"13.397634,52.529407", 200, 300},
		{"13.397634,52.529407", 500, 100},
	}

	var output = Output{
		Source: "13.388860,52.517037",
		Routes: routes,
	}

	output.sortRoutesByDurationAsc()

	result := reflect.DeepEqual(output.Routes, expectedRoutes)

	if !result {
		t.Fatal("Sort order is not equal to", expectedRoutes)
	}
}
