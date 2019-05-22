module github.com/funswe/flow

go 1.12

require (
	github.com/json-iterator/go v1.1.6
	github.com/julienschmidt/httprouter v1.2.0
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/flosch/pongo2 v0.0.0
	golang.org/x/net v0.0.0
	golang.org/x/crypto v0.0.0
	golang.org/x/text v0.0.0
	golang.org/x/tools v0.0.0
)

replace (
	golang.org/x/net v0.0.0 => github.com/golang/net v0.0.0-20190520210107-018c4d40a106
	golang.org/x/crypto v0.0.0 => github.com/golang/crypto v0.0.0-20190513172903-22d7a77e9e5f
	golang.org/x/text v0.0.0 => github.com/golang/text v0.3.2
	golang.org/x/tools v0.0.0 => github.com/golang/tools v0.0.0-20190521203540-521d6ed310dd
)
