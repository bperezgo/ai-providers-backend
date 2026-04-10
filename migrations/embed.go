package migrations

import "embed"

//go:embed sql/*.sql
var SQLFiles embed.FS

// Dir is the directory inside SQLFiles that contains the .sql migration files.
const Dir = "sql"

//go:embed seeds/*.sql
var SeedFiles embed.FS

// SeedDir is the directory inside SeedFiles that contains seed .sql files.
const SeedDir = "seeds"
