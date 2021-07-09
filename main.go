package main

import (
	"log"
	"net/http"
	"strings"
)

// 这句话是关键。。
type HTTPHandler func(*Context)

// 这里就是一个可变数组
type engine struct {
	*Group
	groups []*Group // 所有的groups需要串联起来
}

type Group struct {
	prefix      string // 支持叠加
	router      *router
	middlewares []HTTPHandler
}

func (g *Group) NewGroup(prefix string) *Group {
	newGroup := &Group{prefix: g.prefix + prefix, router: g.router} // 上一级router的方法。
	return newGroup
}

func (g *Group) addRouter(m, p string, h HTTPHandler) {
	pattern := g.prefix + p // 将前缀委托上去
	g.router.addRouter(m, pattern, h)
}

func (g *Group) Get(p string, h HTTPHandler) {
	g.addRouter("GET", p, h)
}

func (g *Group) Post(p string, h HTTPHandler) {
	g.addRouter("POST", p, h)
}

// 保存的上下文
type Context struct {
	w            http.ResponseWriter
	res          *http.Request
	path, method string
	Params       map[string]string
	statusCode   int
	index        int // 中间件的index
	// 中间件
	handlers []HTTPHandler
}

// 路由部分只不过是被抽象出来了
type router struct {
	roots    map[string]*node
	handlers map[string]HTTPHandler
}

func NewContext(w http.ResponseWriter, r *http.Request) *Context {

	return &Context{
		res:    r,
		w:      w,
		path:   r.URL.Path,
		method: r.Method,
		index:  -1,
	}
}

// 下一个中间件
func (c *Context) Next() {
	c.index++
	s := len(c.handlers)
	//  为什么要把指向index在处理？
	for ; c.index < s; c.index++ {
		// 一次性执行完
		c.handlers[c.index](c) // 处理中间件的内容
	}
}

func (gourp *Group) Use(middlewares ...HTTPHandler) {
	gourp.middlewares = append(gourp.middlewares, middlewares...)
}

func NewRouter() *router {
	return &router{handlers: make(map[string]HTTPHandler), roots: make(map[string]*node)}
}

//  这里就有点问题了。
func NewEngine() *engine {
	engine := &engine{}
	engine.Group = &Group{router: NewRouter()}
	engine.groups = []*Group{engine.Group}
	return engine
}

func (r *router) addRouter(m, p string, handler HTTPHandler) {
	// 这里就不是简单的addrouter了，而是将p要解析一下
	parts := parsePattern(p)
	key := m + "%%" + p
	_, ok := r.roots[m]
	if !ok { // 如果没有找到这个node，直接创建node
		r.roots[m] = &node{}
	}
	r.roots[m].insert(p, parts, 0) // 插入树里面去

	r.handlers[key] = handler
}

func (r *router) getRouter(m, p string) (*node, map[string]string) {
	parsePaths := parsePattern(p) // 将path胡渠道
	params := make(map[string]string)
	root, ok := r.roots[m]
	if !ok {
		return nil, nil
	}
	// 找到第一层
	n := root.search(parsePaths, 0)
	if n != nil {
		//  继续解析
		parts := parsePattern(n.pattern) // 整个路由

		for index, part := range parts {
			if part[0] == ':' {
				// 就是模糊查询嘛。。真复杂。。
				// params["s"] = "f"
				params[part[1:]] = parsePaths[index] // 这个是哪里，获取传过来的值。。
			}
			if part[0] == '*' && len(part) > 1 {
				params[part[1:]] = strings.Join(parsePaths[index:], "/")
				break
			}
		}
		return n, params
	}
	return nil, nil
}

func (r *router) handle(c *Context) {
	log.Println(c.method, c.path)
	n, params := r.getRouter(c.method, c.path)
	if n != nil {
		c.Params = params
		key := c.method + "%%" + n.pattern
		//  这里是中间件后面的事情
		c.handlers = append(c.handlers, r.handlers[key]) // 把中间件放在里面
	} else {
		c.handlers = append(c.handlers, func(c *Context) {

		})
	}
	// 下一个中间件
	c.Next()
}

//  将路由里面的
func parsePattern(pattern string) []string {
	vs := strings.Split(pattern, "/")
	parts := make([]string, 0)

	for _, item := range vs {
		if item != "" { // 将所有路径加进去
			parts = append(parts, item)
			if item[0] == '*' { // 如果第一个*的话，不要加进去
				break
			}
		}
	}
	return parts
}

// 表单信息
func (c *Context) PostForm(value string) string {
	return c.res.FormValue(value)
}

// 查找里面的数据
func (c *Context) Query(value string) string {
	return c.res.URL.Query().Get(value)
}

func (c *Context) Param(key string) string {
	value, _ := c.Params[key]
	return value
}

// 添加路由
func (e *engine) GET(path string, HTTPHandler HTTPHandler) {
	e.router.addRouter("GET", path, HTTPHandler)
}

func (e *engine) POST(path string, handler HTTPHandler) {
	e.router.addRouter("POST", path, handler)
}

func (e *engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	context := NewContext(w, r)

	var ms []HTTPHandler
	// 这里是处理group的路由
	for _, group := range e.groups {
		if strings.HasPrefix(r.URL.Path, group.prefix) {
			ms = append(ms, group.middlewares...)
		}
	}
	// 传入空的路由中间件
	for i := range ms {
		// 这里没有路由
		log.Println("===>ms", i)
	}
	context.handlers = ms
	e.router.handle(context)
}

func (e *engine) Run(addr string) error {
	log.Println("Run on ", addr)
	return http.ListenAndServe(addr, e)
}

func Logger() HTTPHandler {
	return func(c *Context) {
		log.Println("hook 1 start")
		// c.Next() // 我猜到达不了下一个中间件
		// 为啥这个可以执行？
		log.Println("hook 1 end") // 直到这个中间
	}
}

func Logger2() HTTPHandler {
	return func(c *Context) {
		log.Println("hook 2 start")
		// c.Next() // 我猜到达不了下一个中间件  这个是个Bug
		// 为啥这个可以执行？
		log.Println("hook 2 end") // 直到这个中间
	}
}

func main() {
	r := NewEngine()
	s := r.NewGroup("/fn")
	s.Get("/curl", func(c *Context) {
		c.w.Write([]byte("func"))
		//  s继承 gourp 和继承了 enigne
	})
	r.Use(Logger())
	r.Use(Logger2())
	r.GET("/", func(ctx *Context) {
		ctx.w.Write([]byte("ok"))
	})
	r.GET("/func", func(c *Context) {
		c.w.Write([]byte("func"))
	})
	r.GET("/f/:s", func(c *Context) {
		c.w.Write([]byte("s"))
	})

	r.Run(":9090")
}
