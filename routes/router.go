package routes

import (
	"fmt"
	"net/http"
	"strings"
	"io"
	"bytes"
)

type Handle func(http.ResponseWriter, *http.Request, Params)

func convertToText(reader io.Reader) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	return buf.String()
}

var NotFoundHandle = func() Handle {
	return func(w http.ResponseWriter, r *http.Request, p Params) {
		notfoundPreHandler(r, p)
		w.WriteHeader(404)
		fmt.Fprintf(w, "404 NotFound")
	}
}()
var notfoundPreHandlerForProduction = func(r *http.Request, p Params) {}
var notfoundPreHandlerForDebug = func(r *http.Request, p Params) {
	fmt.Println(r.URL.Path, r.Method, p, convertToText(r.Body))
}
var notfoundPreHandler = notfoundPreHandlerForProduction
var debugMode bool = false
var instance *Router
var NotFoundNode = createNotFoundNode()
var NotFoundLeaf = createLeaf(NotFoundNode, "", NotFoundHandle)

type Router struct {
	root *RouteNode
}
type Params struct {
	values map[string]string
}
type RouteResult struct {
	leaf   *RouteLeaf
	params *Params
}
type RouteNode struct {
	path         string
	method       string
	originPath   string
	level        int8
	nodes        map[string]*RouteNode
	leaves       map[string]*RouteLeaf
	concateParam bool
}
type RouteLeaf struct {
	node   *RouteNode
	method string
	handle Handle
}

func (p *Params) Get(field string) string {
	return p.values[field]
}

func GetInstance() *Router {
	if instance == nil {
		instance = new(Router)
		instance.root = createNode("", "", "")
	}
	return instance
}

func (router *Router) traverseNode(path string, method string) *RouteResult {
	arr := strings.Split(path, "/")[1:]
	var node = router.root
	params := createParams()
	for _, name := range arr {
		var n1 = node.nodes[name]
		if n1 == nil {
			var n2 = node.nodes[".*"]
			if n2 == nil {
				node = NotFoundNode
				break
			} else {
				if n2.concateParam {
					params.values[n2.originPath] += "/" + name
				} else {
					params.values[n2.originPath] = name
				}
				node = n2
			}
		} else {
			node = n1
		}
	}
	leaf := node.findLeaf(method)
	return createRouteResult(leaf, params)
}

func nameToPattern(name string) string {
	if name[0] == ':' || name[0] == '*' {
		return ".*"
	} else {
		return name
	}
}

func SetDebugMode(isDebugMode bool) {
	debugMode = isDebugMode
	if debugMode {
		notfoundPreHandler = notfoundPreHandlerForDebug
	} else {
		notfoundPreHandler = notfoundPreHandlerForProduction
	}
}

func (router *Router) Handle(path string, method string, handle Handle) {
	arr := strings.Split(path, "/")[1:]
	var lastNode = router.root
	lenOfArr := len(arr)
	for idx := 0; idx < lenOfArr; idx++ {
		name := arr[idx]
		pName := nameToPattern(name)
		node := lastNode.popupNodeIfNotExists(name, pName)
		lastNode = node
		if name[0] == '*' {
			lastNode.add(lastNode)
			break
		}
	}
	lastNode.addLeaf(method, handle)
}

func (router *Router) Handler(path string, method string, handler http.Handler) {
	router.Handle(path, method, func(w http.ResponseWriter, r *http.Request, _ Params) {
		handler.ServeHTTP(w, r)
	})
}

func (router *Router) HandlerFunc(path string, method string, handler http.HandlerFunc) {
	router.HandlerFunc(path, method, handler)
}

func (router *Router) GET(path string, handle Handle) {
	router.Handle(path, "GET", handle)
}

func (router *Router) PUT(path string, handle Handle) {
	router.Handle(path, "PUT", handle)
}

func (router *Router) POST(path string, handle Handle) {
	router.Handle(path, "POST", handle)
}

func (router *Router) DELETE(path string, handle Handle) {
	router.Handle(path, "DELETE", handle)
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	routeResult := router.traverseNode(r.URL.Path, r.Method)
	routeResult.leaf.handle(w, r, *(routeResult.params))
}

func (router *Router) ServeFiles(dirName string, logicalPath string) {
	fs := http.Dir(logicalPath)
	fileServer := http.FileServer(fs)
	router.GET(dirName+"/*name", func(w http.ResponseWriter, r *http.Request, p Params) {
		r.URL.Path = p.Get("name")
		fileServer.ServeHTTP(w, r)
	})
}

func (node *RouteNode) addLeaf(method string, handle Handle) {
	node.leaves[method] = createLeaf(node, method, handle)
}

func (node *RouteNode) findLeaf(method string) *RouteLeaf {
	leaf := node.leaves[method]
	if leaf == nil {
		return NotFoundLeaf
	} else {
		return leaf
	}
}

func (node *RouteNode) add(child *RouteNode) {
	node.nodes[child.path] = child
}

func (node *RouteNode) popupNodeIfNotExists(name string, pattern string) *RouteNode {
	if node.nodes[pattern] == nil {
		newNode := popupNode(name)
		newNode.level = node.level + 1
		node.add(newNode)
		return newNode
	} else {
		return node.nodes[pattern]
	}
}

func createNode(path string, method string, originPath string) *RouteNode {
	return &RouteNode{
		path:         path,
		originPath:   originPath,
		method:       method,
		level:        0,
		nodes:        make(map[string]*RouteNode),
		leaves:       make(map[string]*RouteLeaf),
		concateParam: false,
	}
}

func createNotFoundNode() *RouteNode {
	node := createNode("", "", "")
	node.addLeaf("GET", NotFoundHandle)
	node.addLeaf("PUT", NotFoundHandle)
	node.addLeaf("POST", NotFoundHandle)
	node.addLeaf("DELETE", NotFoundHandle)
	return node
}

func createLeaf(node *RouteNode, method string, handle Handle) *RouteLeaf {
	return &RouteLeaf{
		node:   node,
		method: method,
		handle: handle,
	}
}

func createParams() *Params {
	return &Params{
		values: make(map[string]string),
	}
}

func createRouteResult(leaf *RouteLeaf, params *Params) *RouteResult {
	return &RouteResult{
		leaf:   leaf,
		params: params,
	}
}

func popupNode(value string) *RouteNode {
	var idx int
	idx = -1
	var buff string
	var pathRegexp string
	pathRegexp = value
	var concateParam = false
	for i := 0; i < len(value); i++ {
		val := value[i]
		if val == ':' {
			idx = i
		} else if val == '*' {
			concateParam = true
			idx = i
		}
	}
	if idx >= 0 {
		buff = value[idx+1:]
		pathRegexp = strings.Replace(pathRegexp, "*"+buff, ".*", -1)
		pathRegexp = strings.Replace(pathRegexp, ":"+buff, ".*", -1)
		idx = -1
	}
	node := createNode(pathRegexp, "", buff)
	node.concateParam = concateParam
	return node
}
