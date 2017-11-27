// package webx provides a restful API server.

/*
webx provides a restful API server by golang that can debug, build and host a SPA(single page appliaction).


APIService

APIService provides a restful API service with user privilege.

	var apis = &webx.APIService {
		Prefix: 'v2',
	}

	apis.Options('*', webx.PublicCORS)
	apis.Head('endpoint', func(ctx *webx.Context) {}, 'privilegeId')
	apis.Get('endpoint', func(ctx *webx.Context) {}, 'privilegeId')
	apis.POST('endpoint', func(ctx *webx.Context) {}, 'privilegeId')
	apis.Put('endpoint', func(ctx *webx.Context) {}, 'privilegeId')
	apis.Patch('endpoint', func(ctx *webx.Context) {}, 'privilegeId')
	apis.Delete('endpoint', func(ctx *webx.Context) {}, 'privilegeId')

Context

Context...


*/
package webx
