There are 3 prefixes for .go files in this
task module http layer source files:
1. 0-handler.go: handler struct and route registration
2. 1-*: http handler implementations
3. 2-*: helper functions for http handlers

Handler struct basically defines interfaces, caches
and mutexes accessible to http handlers, and
`NewTaskHttpHandler` is the constructor function.

