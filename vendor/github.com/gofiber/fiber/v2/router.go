// ⚡️ Fiber is an Express inspired web framework written in Go with ☕️
// 🤖 Github Repository: https://github.com/gofiber/fiber
// 📌 API Documentation: https://docs.gofiber.io

package fiber

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2/utils"

	"github.com/valyala/fasthttp"
)

// Router defines all router handle interface, including app and group router.
type Router interface {
	Use(args ...interface{}) Router

	Get(path string, handlers ...Handler) Router
	Head(path string, handlers ...Handler) Router
	Post(path string, handlers ...Handler) Router
	Put(path string, handlers ...Handler) Router
	Delete(path string, handlers ...Handler) Router
	Connect(path string, handlers ...Handler) Router
	Options(path string, handlers ...Handler) Router
	Trace(path string, handlers ...Handler) Router
	Patch(path string, handlers ...Handler) Router

	Add(method, path string, handlers ...Handler) Router
	Static(prefix, root string, config ...Static) Router
	All(path string, handlers ...Handler) Router

	Group(prefix string, handlers ...Handler) Router

	Route(prefix string, fn func(router Router), name ...string) Router

	Mount(prefix string, fiber *App) Router

	Name(name string) Router
}

// Route is a struct that holds all metadata for each registered handler.
type Route struct {
	// ### important: always keep in sync with the copy method "app.copyRoute" and all creations of Route struct ###
	// Data for routing
	use         bool        // USE matches path prefixes
	mount       bool        // Indicated a mounted app on a specific route
	star        bool        // Path equals '*'
	root        bool        // Path equals '/'
	path        string      // Prettified path
	routeParser routeParser // Parameter parser
	group       *Group      // Group instance. used for routes in groups

	// Public fields
	Method string `json:"method"` // HTTP method
	Name   string `json:"name"`   // Route's name
	//nolint:revive // Having both a Path (uppercase) and a path (lowercase) is fine
	Path     string    `json:"path"`   // Original registered route path
	Params   []string  `json:"params"` // Case sensitive param keys
	Handlers []Handler `json:"-"`      // Ctx handlers
}

func (r *Route) match(detectionPath, path string, params *[maxParams]string) bool {
	// root detectionPath check
	if r.root && detectionPath == "/" {
		return true
		// '*' wildcard matches any detectionPath
	} else if r.star {
		if len(path) > 1 {
			params[0] = path[1:]
		} else {
			params[0] = ""
		}
		return true
	}
	// Does this route have parameters
	if len(r.Params) > 0 {
		// Match params
		if match := r.routeParser.getMatch(detectionPath, path, params, r.use); match {
			// Get params from the path detectionPath
			return match
		}
	}
	// Is this route a Middleware?
	if r.use {
		// Single slash will match or detectionPath prefix
		if r.root || strings.HasPrefix(detectionPath, r.path) {
			return true
		}
		// Check for a simple detectionPath match
	} else if len(r.path) == len(detectionPath) && r.path == detectionPath {
		return true
	}
	// No match
	return false
}

func (app *App) next(c *Ctx) (bool, error) {
	// Get stack length
	tree, ok := app.treeStack[c.methodINT][c.treePath]
	if !ok {
		tree = app.treeStack[c.methodINT][""]
	}
	lenTree := len(tree) - 1

	// Loop over the route stack starting from previous index
	for c.indexRoute < lenTree {
		// Increment route index
		c.indexRoute++

		// Get *Route
		route := tree[c.indexRoute]

		var match bool
		var err error
		// skip for mounted apps
		if route.mount {
			continue
		}

		// Check if it matches the request path
		match = route.match(c.detectionPath, c.path, &c.values)
		if !match {
			// No match, next route
			continue
		}
		// Pass route reference and param values
		c.route = route

		// Non use handler matched
		if !c.matched && !route.use {
			c.matched = true
		}

		// Execute first handler of route
		c.indexHandler = 0
		if len(route.Handlers) > 0 {
			err = route.Handlers[0](c)
		}
		return match, err // Stop scanning the stack
	}

	// If c.Next() does not match, return 404
	err := NewError(StatusNotFound, "Cannot "+c.method+" "+html.EscapeString(c.pathOriginal))
	if !c.matched && app.methodExist(c) {
		// If no match, scan stack again if other methods match the request
		// Moved from app.handler because middleware may break the route chain
		err = ErrMethodNotAllowed
	}
	return false, err
}

func (app *App) handler(rctx *fasthttp.RequestCtx) { //revive:disable-line:confusing-naming // Having both a Handler() (uppercase) and a handler() (lowercase) is fine. TODO: Use nolint:revive directive instead. See https://github.com/golangci/golangci-lint/issues/3476
	// Acquire Ctx with fasthttp request from pool
	c := app.AcquireCtx(rctx)
	defer app.ReleaseCtx(c)

	// handle invalid http method directly
	if c.methodINT == -1 {
		_ = c.Status(StatusBadRequest).SendString("Invalid http method") //nolint:errcheck // It is fine to ignore the error here
		return
	}

	// Find match in stack
	match, err := app.next(c)
	if err != nil {
		if catch := c.app.ErrorHandler(c, err); catch != nil {
			_ = c.SendStatus(StatusInternalServerError) //nolint:errcheck // It is fine to ignore the error here
		}
		// TODO: Do we need to return here?
	}
	// Generate ETag if enabled
	if match && app.config.ETag {
		setETag(c, false)
	}
}

func (app *App) addPrefixToRoute(prefix string, route *Route) *Route {
	prefixedPath := getGroupPath(prefix, route.Path)
	prettyPath := prefixedPath
	// Case-sensitive routing, all to lowercase
	if !app.config.CaseSensitive {
		prettyPath = utils.ToLower(prettyPath)
	}
	// Strict routing, remove trailing slashes
	if !app.config.StrictRouting && len(prettyPath) > 1 {
		prettyPath = utils.TrimRight(prettyPath, '/')
	}

	route.Path = prefixedPath
	route.path = RemoveEscapeChar(prettyPath)
	route.routeParser = parseRoute(prettyPath)
	route.root = false
	route.star = false

	return route
}

func (*App) copyRoute(route *Route) *Route {
	return &Route{
		// Router booleans
		use:   route.use,
		mount: route.mount,
		star:  route.star,
		root:  route.root,

		// Path data
		path:        route.path,
		routeParser: route.routeParser,

		// Public data
		Path:     route.Path,
		Params:   route.Params,
		Name:     route.Name,
		Method:   route.Method,
		Handlers: route.Handlers,
	}
}

func (app *App) register(method, pathRaw string, group *Group, handlers ...Handler) {
	// Uppercase HTTP methods
	method = utils.ToUpper(method)
	// Check if the HTTP method is valid unless it's USE
	if method != methodUse && app.methodInt(method) == -1 {
		panic(fmt.Sprintf("add: invalid http method %s\n", method))
	}
	// is mounted app
	isMount := group != nil && group.app != app
	// A route requires atleast one ctx handler
	if len(handlers) == 0 && !isMount {
		panic(fmt.Sprintf("missing handler in route: %s\n", pathRaw))
	}
	// Cannot have an empty path
	if pathRaw == "" {
		pathRaw = "/"
	}
	// Path always start with a '/'
	if pathRaw[0] != '/' {
		pathRaw = "/" + pathRaw
	}
	// Create a stripped path in-case sensitive / trailing slashes
	pathPretty := pathRaw
	// Case-sensitive routing, all to lowercase
	if !app.config.CaseSensitive {
		pathPretty = utils.ToLower(pathPretty)
	}
	// Strict routing, remove trailing slashes
	if !app.config.StrictRouting && len(pathPretty) > 1 {
		pathPretty = utils.TrimRight(pathPretty, '/')
	}
	// Is layer a middleware?
	isUse := method == methodUse
	// Is path a direct wildcard?
	isStar := pathPretty == "/*"
	// Is path a root slash?
	isRoot := pathPretty == "/"
	// Parse path parameters
	parsedRaw := parseRoute(pathRaw)
	parsedPretty := parseRoute(pathPretty)

	// Create route metadata without pointer
	route := Route{
		// Router booleans
		use:   isUse,
		mount: isMount,
		star:  isStar,
		root:  isRoot,

		// Path data
		path:        RemoveEscapeChar(pathPretty),
		routeParser: parsedPretty,
		Params:      parsedRaw.params,

		// Group data
		group: group,

		// Public data
		Path:     pathRaw,
		Method:   method,
		Handlers: handlers,
	}
	// Increment global handler count
	atomic.AddUint32(&app.handlersCount, uint32(len(handlers)))

	// Middleware route matches all HTTP methods
	if isUse {
		// Add route to all HTTP methods stack
		for _, m := range app.config.RequestMethods {
			// Create a route copy to avoid duplicates during compression
			r := route
			app.addRoute(m, &r)
		}
	} else {
		// Add route to stack
		app.addRoute(method, &route)
	}
}

func (app *App) registerStatic(prefix, root string, config ...Static) {
	// For security, we want to restrict to the current work directory.
	if root == "" {
		root = "."
	}
	// Cannot have an empty prefix
	if prefix == "" {
		prefix = "/"
	}
	// Prefix always start with a '/' or '*'
	if prefix[0] != '/' {
		prefix = "/" + prefix
	}
	// in case-sensitive routing, all to lowercase
	if !app.config.CaseSensitive {
		prefix = utils.ToLower(prefix)
	}
	// Strip trailing slashes from the root path
	if len(root) > 0 && root[len(root)-1] == '/' {
		root = root[:len(root)-1]
	}
	// Is prefix a direct wildcard?
	isStar := prefix == "/*"
	// Is prefix a root slash?
	isRoot := prefix == "/"
	// Is prefix a partial wildcard?
	if strings.Contains(prefix, "*") {
		// /john* -> /john
		isStar = true
		prefix = strings.Split(prefix, "*")[0]
		// Fix this later
	}
	prefixLen := len(prefix)
	if prefixLen > 1 && prefix[prefixLen-1:] == "/" {
		// /john/ -> /john
		prefixLen--
		prefix = prefix[:prefixLen]
	}
	const cacheDuration = 10 * time.Second
	// Fileserver settings
	fs := &fasthttp.FS{
		Root:                 root,
		AllowEmptyRoot:       true,
		GenerateIndexPages:   false,
		AcceptByteRange:      false,
		Compress:             false,
		CompressedFileSuffix: app.config.CompressedFileSuffix,
		CacheDuration:        cacheDuration,
		IndexNames:           []string{"index.html"},
		PathRewrite: func(fctx *fasthttp.RequestCtx) []byte {
			path := fctx.Path()
			if len(path) >= prefixLen {
				if isStar && app.getString(path[0:prefixLen]) == prefix {
					path = append(path[0:0], '/')
				} else {
					path = path[prefixLen:]
					if len(path) == 0 || path[len(path)-1] != '/' {
						path = append(path, '/')
					}
				}
			}
			if len(path) > 0 && path[0] != '/' {
				path = append([]byte("/"), path...)
			}
			return path
		},
		PathNotFound: func(fctx *fasthttp.RequestCtx) {
			fctx.Response.SetStatusCode(StatusNotFound)
		},
	}

	// Set config if provided
	var cacheControlValue string
	var modifyResponse Handler
	if len(config) > 0 {
		maxAge := config[0].MaxAge
		if maxAge > 0 {
			cacheControlValue = "public, max-age=" + strconv.Itoa(maxAge)
		}
		fs.CacheDuration = config[0].CacheDuration
		fs.Compress = config[0].Compress
		fs.AcceptByteRange = config[0].ByteRange
		fs.GenerateIndexPages = config[0].Browse
		if config[0].Index != "" {
			fs.IndexNames = []string{config[0].Index}
		}
		modifyResponse = config[0].ModifyResponse
	}
	fileHandler := fs.NewRequestHandler()
	handler := func(c *Ctx) error {
		// Don't execute middleware if Next returns true
		if len(config) != 0 && config[0].Next != nil && config[0].Next(c) {
			return c.Next()
		}
		// Serve file
		fileHandler(c.fasthttp)
		// Sets the response Content-Disposition header to attachment if the Download option is true
		if len(config) > 0 && config[0].Download {
			c.Attachment()
		}
		// Return request if found and not forbidden
		status := c.fasthttp.Response.StatusCode()
		if status != StatusNotFound && status != StatusForbidden {
			if len(cacheControlValue) > 0 {
				c.fasthttp.Response.Header.Set(HeaderCacheControl, cacheControlValue)
			}
			if modifyResponse != nil {
				return modifyResponse(c)
			}
			return nil
		}
		// Reset response to default
		c.fasthttp.SetContentType("") // Issue #420
		c.fasthttp.Response.SetStatusCode(StatusOK)
		c.fasthttp.Response.SetBodyString("")
		// Next middleware
		return c.Next()
	}

	// Create route metadata without pointer
	route := Route{
		// Router booleans
		use:   true,
		mount: false,
		star:  isStar,
		root:  isRoot,

		// Path data
		path: prefix,

		// Group data
		group: nil,

		// Public data
		Path:     prefix,
		Method:   MethodGet,
		Handlers: []Handler{handler},
	}
	// Increment global handler count
	atomic.AddUint32(&app.handlersCount, 1)
	// Add route to stack
	app.addRoute(MethodGet, &route)
	// Add HEAD route
	app.addRoute(MethodHead, &route)
}

func (app *App) addRoute(method string, route *Route) {
	// Get unique HTTP method identifier
	m := app.methodInt(method)

	// prevent identically route registration
	l := len(app.stack[m])
	if l > 0 && app.stack[m][l-1].Path == route.Path && route.use == app.stack[m][l-1].use && !route.mount && !app.stack[m][l-1].mount {
		preRoute := app.stack[m][l-1]
		preRoute.Handlers = append(preRoute.Handlers, route.Handlers...)
	} else {
		route.Method = method
		// Add route to the stack
		app.stack[m] = append(app.stack[m], route)
		app.routesRefreshed = true
	}

	// Execute onRoute hooks & change latestRoute if not adding mounted route
	if !route.mount {
		app.mutex.Lock()
		app.latestRoute = route
		if err := app.hooks.executeOnRouteHooks(*route); err != nil {
			panic(err)
		}
		app.mutex.Unlock()
	}
}

// buildTree build the prefix tree from the previously registered routes
func (app *App) buildTree() *App {
	// If routes haven't been refreshed, nothing to do
	if !app.routesRefreshed {
		return app
	}

	// 1) First loop: determine all possible 3-char prefixes ("treePaths") for each method
	for method := range app.config.RequestMethods {
		prefixSet := map[string]struct{}{
			"": {},
		}
		for _, route := range app.stack[method] {
			if len(route.routeParser.segs) > 0 && len(route.routeParser.segs[0].Const) >= 3 {
				prefix := route.routeParser.segs[0].Const[:3]
				prefixSet[prefix] = struct{}{}
			}
		}
		tsMap := make(map[string][]*Route, len(prefixSet))
		for prefix := range prefixSet {
			tsMap[prefix] = nil
		}
		app.treeStack[method] = tsMap
	}

	// 2) Second loop: for each method and each discovered treePath, assign matching routes
	for method := range app.config.RequestMethods {
		// get the map of buckets for this method
		tsMap := app.treeStack[method]

		// for every treePath key (including the empty one)
		for treePath := range tsMap {
			// iterate all routes of this method
			for _, route := range app.stack[method] {
				// compute this route's own prefix ("" or first 3 chars)
				routePath := ""
				if len(route.routeParser.segs) > 0 && len(route.routeParser.segs[0].Const) >= 3 {
					routePath = route.routeParser.segs[0].Const[:3]
				}

				// if it's a global route, assign to every bucket
				if routePath == "" {
					tsMap[treePath] = append(tsMap[treePath], route)
					// otherwise only assign if this route's prefix matches the current bucket's key
				} else if routePath == treePath {
					tsMap[treePath] = append(tsMap[treePath], route)
				}
			}

			// after collecting, dedupe the bucket if it's not the global one
			tsMap[treePath] = uniqueRouteStack(tsMap[treePath])
		}
	}

	// reset the flag and return
	app.routesRefreshed = false
	return app
}
