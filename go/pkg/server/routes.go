package server

import "net/http"

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type Routes []Route

var routes = Routes{
	Route{
		"Process",
		"POST",
		"/process",
		Process,
	},
	Route{
		"UploadFile",
		"POST",
		"/upload",
		UploadFile,
	},
}
