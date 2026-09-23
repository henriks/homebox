package main

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/migrations"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
)

func TestScaleImagesMigrationUpgradesExistingSQLiteGroup(t *testing.T) {
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "upgrade.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	files, err := migrations.Migrations(config.DriverSqlite3)
	require.NoError(t, err)
	goose.SetBaseFS(files)
	require.NoError(t, goose.SetDialect(config.DriverSqlite3))
	require.NoError(t, goose.UpTo(db, config.DriverSqlite3, 20260512130000))

	_, err = db.Exec(`INSERT INTO groups (id, created_at, updated_at, name, currency) VALUES (?, ?, ?, ?, ?)`,
		"00000000-0000-0000-0000-000000000001", time.Now(), time.Now(), "existing", "usd")
	require.NoError(t, err)

	client := ent.NewClient(ent.Driver(entsql.OpenDB("sqlite3", db)))
	require.NoError(t, runMigrations(client, config.DriverSqlite3))

	var enabled bool
	require.NoError(t, db.QueryRow(`SELECT scale_images FROM groups WHERE name = ?`, "existing").Scan(&enabled))
	require.False(t, enabled)
}
