package sermo

import (
	"regexp"
	"sort"
	"strings"
)

type Request struct {
	Method  string                 `json:"method"`
	URL     string                 `json:"url"`
	Query   map[string]interface{} `json:"query"`
	Body    map[string]interface{} `json:"body"`
	Headers map[string]interface{} `json:"headers"`
	Params  map[string]interface{} `json:"params"`

	RequestID string `json:"requestId"`
}

type Response struct {
	Type string      `json:"type"`
	URL  string      `json:"url"`
	Code int         `json:"code"`
	Data interface{} `json:"data"`

	RequestID string `json:"requestId"`
	Client    Client `json:"-"`
}

func (r Response) Send(res Response) (int, error) {
	res.Type = r.Type
	res.URL = r.URL
	res.RequestID = r.RequestID
	return r.Client.Write(res)
}

type Route struct {
	Method        string
	Version       string
	URL           string
	Params        []string
	RouteFunction func(req Request, res Response) (int, error)
}
type Routes []Route

func (r *Routes) Get(version string, url string, routeFunction func(req Request, res Response) (int, error)) {
	r.RegisterRoute("get", version, url, routeFunction)
}

func (r *Routes) Post(version string, url string, routeFunction func(req Request, res Response) (int, error)) {
	r.RegisterRoute("post", version, url, routeFunction)
}

func (r *Routes) Put(version string, url string, routeFunction func(req Request, res Response) (int, error)) {
	r.RegisterRoute("put", version, url, routeFunction)
}

func (r *Routes) Patch(version string, url string, routeFunction func(req Request, res Response) (int, error)) {
	r.RegisterRoute("patch", version, url, routeFunction)
}

func (r *Routes) Delete(version string, url string, routeFunction func(req Request, res Response) (int, error)) {
	r.RegisterRoute("delete", version, url, routeFunction)
}

func (r *Routes) RegisterRoute(method string, version string, url string, routeFunction func(req Request, res Response) (int, error)) {
	route := Route{
		Method:        method,
		Version:       version,
		URL:           "/" + version + url,
		RouteFunction: routeFunction,
	}

	for idx, param := range strings.Split(route.URL, ":") {
		if idx > 0 {
			route.Params = append(route.Params, strings.ReplaceAll(param, "/", ""))
		}
	}

	*r = append(*r, route)

	// Sort routes based on number of parameters
	sort.Slice(*r, func(index, indexNew int) bool {
		route := *r
		return len(route[index].Params) > len(route[indexNew].Params)
	})
}

func (route Route) urlRegex() *regexp.Regexp {
	url := strings.ToLower(route.URL)

	// Get parameters
	for _, param := range route.Params {
		paramRegex := regexp.MustCompile(":" + param)
		url = paramRegex.ReplaceAllString(url, "(.*)")
	}

	return regexp.MustCompile(url)
}
