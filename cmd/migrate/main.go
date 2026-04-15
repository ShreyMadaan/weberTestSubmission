package main

import (
"context"
"fmt"
"io/fs"
"log"
"os"
"path/filepath"
"sort"
"strings"
"time"

"github.com/jackc/pgx/v5"
"github.com/your-username/auctioncore/internal/config"
)

func main() {
cfg := config.Load()

dsn := fmt.Sprintf(
"postgres://%s:%s@%s:%s/%s?sslmode=%s",
cfg.PostgresUser,
cfg.PostgresPass,
cfg.PostgresHost,
cfg.PostgresPort,
cfg.PostgresDB,
cfg.PostgresSSL,
)

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

conn, err := pgx.Connect(ctx, dsn)
if err != nil {
log.Fatalf("connect db: %v", err)
}
defer conn.Close(ctx)

if err := ensureSchemaMigrationsTable(ctx, conn); err != nil {
log.Fatalf("ensure schema_migrations: %v", err)
}

migrations, err := collectMigrations("db/migrations")
if err != nil {
log.Fatalf("collect migrations: %v", err)
}

for _, m := range migrations {
if strings.HasSuffix(m.Name, ".up.sql") {
if err := applyMigration(ctx, conn, m); err != nil {
log.Fatalf("apply migration %s: %v", m.Name, err)
}
}
}

log.Println("migrations applied successfully")
}

type migrationFile struct {
Name string
Path string
}

func ensureSchemaMigrationsTable(ctx context.Context, conn *pgx.Conn) error {
_, err := conn.Exec(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
filename TEXT PRIMARY KEY,
applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`)
return err
}

func collectMigrations(dir string) ([]migrationFile, error) {
var files []migrationFile

err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
if err != nil {
return err
}
if d.IsDir() {
return nil
}
if strings.HasSuffix(d.Name(), ".up.sql") {
files = append(files, migrationFile{
Name: d.Name(),
Path: path,
})
}
return nil
})
if err != nil {
return nil, err
}

sort.Slice(files, func(i, j int) bool {
return files[i].Name < files[j].Name
})

return files, nil
}

func migrationApplied(ctx context.Context, conn *pgx.Conn, name string) (bool, error) {
var exists bool
err := conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE filename = $1)`, name).Scan(&exists)
return exists, err
}

func applyMigration(ctx context.Context, conn *pgx.Conn, m migrationFile) error {
applied, err := migrationApplied(ctx, conn, m.Name)
if err != nil {
return err
}
if applied {
log.Printf("skipping already applied migration %s", m.Name)
return nil
}

sqlBytes, err := os.ReadFile(m.Path)
if err != nil {
return err
}
sql := string(sqlBytes)

log.Printf("applying migration %s", m.Name)

tx, err := conn.Begin(ctx)
if err != nil {
return err
}
defer tx.Rollback(ctx)

if _, err := tx.Exec(ctx, sql); err != nil {
return fmt.Errorf("exec migration %s: %w", m.Name, err)
}

if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (filename) VALUES ($1)`, m.Name); err != nil {
return fmt.Errorf("record migration %s: %w", m.Name, err)
}

if err := tx.Commit(ctx); err != nil {
return err
}

return nil
}
