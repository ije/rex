// Package rex provides provides a REST server that can debug, build, and host a SPA(single page appliaction).

/*
Package rex provides provides a REST server that can debug, build, and host a SPA(single page appliaction).


APIService

APIService provides a REST API service with user privileges.

	var apis = &rex.APIService {
		Prefix: 'v2',
	}

	apis.Options('*', rex.PublicCORS)
	apis.Head('endpoint', func(ctx *rex.Context) {}, 'privilegeId')
	apis.Get('endpoint', func(ctx *rex.Context) {}, 'privilegeId')
	apis.POST('endpoint', func(ctx *rex.Context) {}, 'privilegeId')
	apis.Put('endpoint', func(ctx *rex.Context) {}, 'privilegeId')
	apis.Patch('endpoint', func(ctx *rex.Context) {}, 'privilegeId')
	apis.Delete('endpoint', func(ctx *rex.Context) {}, 'privilegeId')
	apis.Use(func(ctx *rex.Context, next func()) {} )

*/
package rex
