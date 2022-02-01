module github.com/felixge/go-prof-app

go 1.16

require (
	github.com/DataDog/datadog-go v4.8.2+incompatible
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/jackc/pgx/v4 v4.13.0
	github.com/julienschmidt/httprouter v1.3.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.8 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	gopkg.in/DataDog/dd-trace-go.v1 v1.36.0
)

// replace gopkg.in/DataDog/dd-trace-go.v1 => /Users/felix.geisendoerfer/go/src/github.com/DataDog/dd-trace-go
