module github.com/mrz1836/go-datastore

go 1.19

require (
	github.com/99designs/gqlgen v0.17.43
	github.com/iancoleman/strcase v0.3.0
	github.com/mrz1836/go-logger v0.3.2
	github.com/newrelic/go-agent/v3 v3.29.1
	github.com/newrelic/go-agent/v3/integrations/nrmongo v1.1.3
	github.com/stretchr/testify v1.8.4
	go.mongodb.org/mongo-driver v1.13.1
	gorm.io/driver/mysql v1.5.3
	gorm.io/driver/postgres v1.5.6
	gorm.io/driver/sqlite v1.5.4
	gorm.io/gorm v1.25.7-0.20240204074919-46816ad31dde
	gorm.io/plugin/dbresolver v1.5.0
)

// Issue: https://github.com/go-gorm/sqlite/issues/168
replace gorm.io/driver/sqlite v1.5.4 => gorm.io/driver/sqlite v1.5.3

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-sql-driver/mysql v1.7.1 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20231201235250-de7065d80cb9 // indirect
	github.com/jackc/pgx/v5 v5.5.2 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/compress v1.17.5 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.20 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/sosodev/duration v1.2.0 // indirect
	github.com/vektah/gqlparser/v2 v2.5.11 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a // indirect
	golang.org/x/crypto v0.18.0 // indirect
	golang.org/x/net v0.20.0 // indirect
	golang.org/x/sync v0.6.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240125205218-1f4bbc51befe // indirect
	google.golang.org/grpc v1.61.0 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
