module ndb

go 1.23

require (
	github.com/gocql/gocql v1.6.0
	github.com/samber/slog-common v0.17.1
)

require (
	github.com/golang/snappy v0.0.3 // indirect
	github.com/hailocab/go-hostpool v0.0.0-20160125115350-e80d13ce29ed // indirect
	github.com/jonboulle/clockwork v0.4.0 // indirect
	github.com/samber/lo v1.44.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
)

replace github.com/gocql/gocql => github.com/scylladb/gocql v1.14.4
