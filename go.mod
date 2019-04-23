module github.com/ije/rex

go 1.12

require (
	github.com/ije/gox v0.1.7
	github.com/julienschmidt/httprouter v1.2.0
	golang.org/x/crypto v0.0.0-20190418165655-df01cb2cc480
)

replace golang.org/x/crypto v0.0.0-20190418165655-df01cb2cc480 => github.com/golang/crypto v0.0.0-20190418165655-df01cb2cc480

replace golang.org/x/sys v0.0.0-20190402142545-baf5eb976a8c => github.com/golang/sys v0.0.0-20190402142545-baf5eb976a8c

replace golang.org/x/sys v0.0.0-20190403152447-81d4e9dc473e => github.com/golang/sys v0.0.0-20190403152447-81d4e9dc473e
