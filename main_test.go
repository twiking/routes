package main

import (
	"reflect"
	"testing"
)

func TestSortRoutes(t *testing.T) {
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
