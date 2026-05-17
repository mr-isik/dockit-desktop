package postgres

import (
	"context"
	"dockit-desktop/internal/domain"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresManager, DbManagerPort interface'ini PostgreSQL için implemente eder.
type PostgresManager struct {
	pool *pgxpool.Pool
}

// NewPostgresManager, boş bir PostgresManager oluşturur.
// Bağlantı açmaz; bağlantı için Connect() metodunu kullanın.
func NewPostgresManager() *PostgresManager {
	return &PostgresManager{}
}

// Connect, verilen DbConnection ile PostgreSQL'e bağlanır.
func (p *PostgresManager) Connect(ctx context.Context, conn domain.DbConnection) error {
	// Önce varsa eski bağlantıyı kapat
	if p.pool != nil {
		p.pool.Close()
	}

	sslMode := strings.TrimSpace(conn.SSLMode)
	if sslMode == "" {
		sslMode = "prefer"
	}

	dsn := buildPostgresDSN(conn, sslMode)

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("bağlantı konfigürasyonu hatalı: %w", err)
	}

	cfg.MaxConns = 5

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("veritabanına bağlanılamadı: %w", err)
	}

	// Bağlantıyı doğrula
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("veritabanına ping atılamadı: %w", err)
	}

	p.pool = pool
	return nil
}

// Disconnect, aktif bağlantı havuzunu kapatır.
func (p *PostgresManager) Disconnect(_ context.Context) error {
	if p.pool != nil {
		p.pool.Close()
		p.pool = nil
	}
	return nil
}

// Ping, bağlantının sağlıklı olup olmadığını test eder.
func (p *PostgresManager) Ping(ctx context.Context) error {
	if p.pool == nil {
		return fmt.Errorf("aktif bir veritabanı bağlantısı yok")
	}
	return p.pool.Ping(ctx)
}

// ListDatabases, sunucudaki kullanıcı veritabanlarını listeler.
func (p *PostgresManager) ListDatabases(ctx context.Context) ([]domain.DbDatabase, error) {
	if err := p.checkConnection(); err != nil {
		return nil, err
	}

	const q = `
SELECT datname, pg_catalog.pg_get_userbyid(datdba) AS owner
FROM pg_catalog.pg_database
WHERE NOT datistemplate
ORDER BY datname`

	rows, err := p.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("veritabanları listelenemedi: %w", err)
	}
	defer rows.Close()

	var result []domain.DbDatabase
	for rows.Next() {
		var db domain.DbDatabase
		if err := rows.Scan(&db.Name, &db.Owner); err != nil {
			return nil, err
		}
		result = append(result, db)
	}
	return result, rows.Err()
}

// ListSchemas, bağlı veritabanındaki kullanıcı şemalarını listeler.
func (p *PostgresManager) ListSchemas(ctx context.Context) ([]domain.DbSchema, error) {
	if err := p.checkConnection(); err != nil {
		return nil, err
	}

	const q = `
SELECT schema_name
FROM information_schema.schemata
WHERE schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
ORDER BY schema_name`

	rows, err := p.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("şemalar listelenemedi: %w", err)
	}
	defer rows.Close()

	var result []domain.DbSchema
	for rows.Next() {
		var s domain.DbSchema
		if err := rows.Scan(&s.Name); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// ListTables, belirtilen şemadaki tabloları ve görünümleri listeler.
func (p *PostgresManager) ListTables(ctx context.Context, schema string) ([]domain.DbTable, error) {
	if err := p.checkConnection(); err != nil {
		return nil, err
	}

	const q = `
SELECT table_schema, table_name, table_type
FROM information_schema.tables
WHERE table_schema = $1
ORDER BY table_name`

	rows, err := p.pool.Query(ctx, q, schema)
	if err != nil {
		return nil, fmt.Errorf("tablolar listelenemedi: %w", err)
	}
	defer rows.Close()

	var result []domain.DbTable
	for rows.Next() {
		var t domain.DbTable
		if err := rows.Scan(&t.Schema, &t.Name, &t.TableType); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

// DescribeTable, bir tablonun sütun bilgilerini ve primary key/unique kısıtlamalarını döndürür.
func (p *PostgresManager) DescribeTable(ctx context.Context, schema, table string) ([]domain.DbColumn, error) {
	if err := p.checkConnection(); err != nil {
		return nil, err
	}

	const q = `
SELECT
    c.column_name,
    c.data_type,
    c.is_nullable = 'YES',
    c.column_default,
    COALESCE(bool_or(tc.constraint_type = 'PRIMARY KEY'), FALSE) AS is_primary_key,
    COALESCE(bool_or(tc.constraint_type = 'UNIQUE'), FALSE) AS is_unique
FROM information_schema.columns c
LEFT JOIN information_schema.key_column_usage kcu
    ON kcu.table_schema = c.table_schema
    AND kcu.table_name = c.table_name
    AND kcu.column_name = c.column_name
LEFT JOIN information_schema.table_constraints tc
    ON tc.constraint_name = kcu.constraint_name
    AND tc.table_schema = kcu.table_schema
    AND tc.table_name = kcu.table_name
WHERE c.table_schema = $1 AND c.table_name = $2
GROUP BY c.column_name, c.data_type, c.is_nullable, c.column_default, c.ordinal_position
ORDER BY c.ordinal_position`

	rows, err := p.pool.Query(ctx, q, schema, table)
	if err != nil {
		return nil, fmt.Errorf("tablo açıklaması alınamadı: %w", err)
	}
	defer rows.Close()

	var result []domain.DbColumn
	for rows.Next() {
		var col domain.DbColumn
		if err := rows.Scan(
			&col.Name,
			&col.DataType,
			&col.IsNullable,
			&col.DefaultValue,
			&col.IsPrimaryKey,
			&col.IsUnique,
		); err != nil {
			return nil, err
		}
		result = append(result, col)
	}
	return result, rows.Err()
}

// ExecuteQuery, ham bir SQL sorgusu çalıştırır.
// SELECT için satırları, DML için etkilenen satır sayısını döndürür.
func (p *PostgresManager) ExecuteQuery(ctx context.Context, query string) (*domain.QueryResult, error) {
	if err := p.checkConnection(); err != nil {
		return nil, err
	}

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		// Sorgu hatasını domain modeline yansıt; Go hatasına dönüştürme
		return &domain.QueryResult{ErrorMessage: err.Error()}, nil
	}
	defer rows.Close()

	// Kolon isimlerini al
	fields := rows.FieldDescriptions()
	colNames := make([]string, len(fields))
	for i, f := range fields {
		colNames[i] = string(f.Name)
	}

	// Tüm satırları topla
	var resultRows [][]interface{}
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		resultRows = append(resultRows, vals)
	}
	if err := rows.Err(); err != nil {
		return &domain.QueryResult{ErrorMessage: err.Error()}, nil
	}

	// pgx, DML için CommandTag üretir
	tag := rows.CommandTag()
	return &domain.QueryResult{
		Columns:      colNames,
		Rows:         resultRows,
		RowsAffected: tag.RowsAffected(),
	}, nil
}

// checkConnection, aktif bir bağlantı olup olmadığını doğrulayan yardımcı metottur.
func (p *PostgresManager) checkConnection() error {
	if p.pool == nil {
		return fmt.Errorf("veritabanına bağlı değilsiniz; önce Connect() çağrısı yapın")
	}
	return nil
}

// collectRows, pgx.Rows'u tek satır slice'ına çeker (yardımcı).
func collectRows[T any](rows pgx.Rows, scanFn func(pgx.CollectableRow) (T, error)) ([]T, error) {
	return pgx.CollectRows(rows, scanFn)
}

func buildPostgresDSN(conn domain.DbConnection, sslMode string) string {
	host := strings.TrimSpace(conn.Host)
	if host == "" {
		host = "localhost"
	}
	port := conn.Port
	if port == 0 {
		port = 5432
	}
	user := strings.TrimSpace(conn.User)
	dbName := strings.TrimSpace(conn.Database)

	var userInfo *url.Userinfo
	if user != "" {
		if conn.Password != "" {
			userInfo = url.UserPassword(user, conn.Password)
		} else {
			userInfo = url.User(user)
		}
	}

	u := &url.URL{
		Scheme: "postgres",
		User:   userInfo,
		Host:   net.JoinHostPort(host, strconv.Itoa(port)),
	}
	if dbName != "" {
		u.Path = "/" + dbName
	}

	q := u.Query()
	if sslMode != "" {
		q.Set("sslmode", sslMode)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
