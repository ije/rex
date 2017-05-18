// webx provides a restful API server that can debug, build and host a SPA(single page appliaction).
/*
webx provides a API server to process data. With nodejs, webx can debug, build and host a SPA(single page appliaction).


APIService

the APIService(map) provides 4 methods(Get,Post,Put,Delete) to create a restful API
map with user privileges, and the API services can be accessed by http request(METHOD /api/endpoint?key=value) after registered

	var apis = webx.APIService{}

	apis.Get('endpoint', func(ctx *webx.Context) {}, user.Privilege_Open)
	apis.Get('endpoint', func(ctx *webx.Context, xs *webx.XService) {}, user.Privilege_Open)
	apis.Get('endpoint', func() {}, user.Privilege_Open)
	apis.Get('endpoint', func() string { return "hello world" }, user.Privilege_Open)
	apis.Get('endpoint', func() (int, string) { return 404, "page not found" }, user.Privilege_Open)
	apis.Post('endpoint', func(ctx *webx.Context) {}, user.Privilege_Open)
	apis.Put('endpoint', func(ctx *webx.Context) {}, user.Privilege_Open)
	apis.Delete('endpoint', func(ctx *webx.Context) {}, user.Privilege_Open)

	webx.Register(apis)


Context

the Context(pointer) is a http request context, as an function argument, is passed by
the api handle.

	var apis = webx.APIService{}

	apis.Get('endpoint', func(ctx *webx.Context) {
		ctx.JSON(200, map[string]interface{
			"words": "hello world",
		})
	}, user.Privilege_Open)

	webx.Register(apis)


XService

the XService(pointer) include some core services(Logging,Session,Users,etc...) of webx,
as an function argument, is passed by the api handle.

	var apis = webx.APIService{}

	apis.Get('endpoint', func(ctx *webx.Context, xs *webx.XService) {
		var session = xs.Session.Get("SID")

		session.Set("KEY", "Value is a interface{}")
	}, user.Privilege_Open)

	webx.Register(apis)
*/
package webx
