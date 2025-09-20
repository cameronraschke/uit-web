module uit-toolbox/api

go 1.25.0

require (
	github.com/jackc/pgx/v5 v5.7.5
	golang.org/x/time v0.12.0
	uit-toolbox/api/database v0.0.0
	uit-toolbox/api/logger v0.0.0
	uit-toolbox/api/post v0.0.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/crypto v0.41.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.29.0 // indirect
)

replace uit-toolbox/api/logger => ./logger

replace uit-toolbox/api/post => ./post

replace uit-toolbox/api/database => ./database
