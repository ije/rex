// package webx provides a restful API server.

/*
webx provides a restful API server by golang that can debug, build and host a SPA(single page appliaction).


APIService

APIService a restful API service with user privilege.

	var apis = &webx.APIService {
		Prefix: 'v2',
	}

	apis.Options('*', webx.PublicCORS)
	apis.Head('endpoint', func(ctx *webx.Context, xs *webx.XService) {}, 'privilegeId')
	apis.Get('endpoint', func(ctx *webx.Context, xs *webx.XService) {}, 'privilegeId')
	apis.POST('endpoint', func(ctx *webx.Context, xs *webx.XService) {}, 'privilegeId')
	apis.Put('endpoint', func(ctx *webx.Context, xs *webx.XService) {}, 'privilegeId')
	apis.Patch('endpoint', func(ctx *webx.Context, xs *webx.XService) {}, 'privilegeId')
	apis.Delete('endpoint', func(ctx *webx.Context, xs *webx.XService) {}, 'privilegeId')

	webx.Register(apis)


Context

Context...


XService

XService...


*/
package webx
