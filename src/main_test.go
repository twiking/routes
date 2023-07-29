package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	router = setupRouter()
)

func mockGetRoutesRequest(url string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	router.ServeHTTP(rec, req)

	return rec
}

func TestGetRoutesReturns400WhenNoParams(t *testing.T) {
	rec := mockGetRoutesRequest("/routes")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetRoutesReturns400WhenNoSrcParam(t *testing.T) {
	rec := mockGetRoutesRequest("/routes?dst=13.397634,52.529407")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetRoutesReturns400WhenNoDstParam(t *testing.T) {
	rec := mockGetRoutesRequest("/routes?src=13.388860,52.517037")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetRoutesReturns400WhenLatLongIsInvalid(t *testing.T) {
	rec := mockGetRoutesRequest("/routes?src=13.388860,52.517037&dst=13.428555,52.523219&dst=invalid")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetRoutesReturns200(t *testing.T) {
	osrmApiPath := "/route/v1/driving/%s;%s"
	src := "13.388860,52.517037"
	attempts := 0
	dst1 := "13.397634,52.529407"
	dst2 := "12.428555,52.523219"
	dst3 := "13.428555,48.523219"
	dst4 := "10.428555,29.523219"

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
			w.Write([]byte(`{"code":"InvalidQuery", "message": "Query string malformed close to position 57"}`))
			return
		}

		p4 := fmt.Sprintf(osrmApiPath, src, dst4)
		if r.URL.Path == p4 && attempts == 0 {
			attempts++
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{}`))
			return
		} else if r.URL.Path == p4 && attempts == 1 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"code":"Ok", "routes": [{"duration":2015.1,"distance":6523.3}]}`))
			return
		}

	}))
	defer mockOsrmApi.Close()

	osrmApiUrl = mockOsrmApi.URL + osrmApiPath
	rec := mockGetRoutesRequest(fmt.Sprintf("/routes?src=%s&dst=%s&dst=%s&dst=%s&dst=%s", src, dst1, dst2, dst3, dst4))

	assert.Equal(t, http.StatusOK, rec.Code)

	expectedResp := `{"source":"13.388860,52.517037","routes":[{"destination":"12.428555,52.523219","duration":260.1,"distance":1886.3},{"destination":"10.428555,29.523219","duration":2015.1,"distance":6523.3},{"destination":"13.397634,52.529407","duration":2490.1,"distance":3286.3}]}`
	assert.Equal(t, expectedResp, rec.Body.String())
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

	var output = GetRoutesResp{
		Source: "13.388860,52.517037",
		Routes: routes,
	}

	output.sortRoutesByDurationAsc()

	result := reflect.DeepEqual(output.Routes, expectedRoutes)

	if !result {
		t.Fatal("Sort order is not equal to", expectedRoutes)
	}
}
