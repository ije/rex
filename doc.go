// package wsx provides a restful API server that can debug, build, and host a SPA(single page appliaction).

/*
WSX provides a restful API server that can debug, build, and host a SPA(single page appliaction).


APIService

APIService provides a restful API service with user privileges.

	var apis = &wsx.APIService {
		Prefix: 'v2',
	}

	apis.Options('*', wsx.PublicCORS)
	apis.Head('endpoint', func(ctx *wsx.Context) {}, 'privilegeId')
	apis.Get('endpoint', func(ctx *wsx.Context) {}, 'privilegeId')
	apis.POST('endpoint', func(ctx *wsx.Context) {}, 'privilegeId')
	apis.Put('endpoint', func(ctx *wsx.Context) {}, 'privilegeId')
	apis.Patch('endpoint', func(ctx *wsx.Context) {}, 'privilegeId')
	apis.Delete('endpoint', func(ctx *wsx.Context) {}, 'privilegeId')

Context

Context...


*/
package wsx
