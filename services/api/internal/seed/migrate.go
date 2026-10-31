package seed

import "context"

// MigrateOnly applies pending goose migrations from dir without touching any data. Production uses it.
func MigrateOnly(ctx context.Context, databaseURL, dir string) error {
	return migrate(ctx, databaseURL, dir)
}
