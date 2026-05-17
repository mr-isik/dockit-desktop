package ports

import (
	"context"
	"dockit-desktop/internal/domain"
)

// EnvStorePort, ortam değişkenlerinin saklanması ve alınması için gerekli
// kalıcı depolama operasyonlarını soyutlar. Open/Closed prensibine göre
// farklı depolama backend'leri (SQLite, Postgres vb.) implemente edebilir.
type EnvStorePort interface {
	// --- Ortam Operasyonları ---

	// CreateEnvironment, yeni bir ortam kaydeder.
	CreateEnvironment(ctx context.Context, env *domain.Environment) error

	// ListEnvironments, tüm kayıtlı ortamları listeler.
	// Değişkenler dahil edilmez; GetVariables ile ayrıca çekilmelidir.
	ListEnvironments(ctx context.Context) ([]domain.Environment, error)

	// DeleteEnvironment, belirtilen ortamı ve tüm değişkenlerini siler.
	DeleteEnvironment(ctx context.Context, id string) error

	// SetActiveEnvironment, belirtilen ortamı aktif yapar (diğerlerini pasif eder).
	SetActiveEnvironment(ctx context.Context, id string) error

	// GetActiveEnvironmentID, aktif ortamın ID'sini döndürür.
	// Aktif ortam yoksa boş string döner.
	GetActiveEnvironmentID(ctx context.Context) (string, error)

	// --- Değişken Operasyonları ---

	// AddVariable, ortama yeni bir değişken ekler.
	// value parametresi ŞIFRESIZ düz metin olarak verilir;
	// implementasyon şifreleme sorumluluğunu DIŞARIDA (usecase/crypto) bırakır.
	// Dolayısıyla store'a gelmeden önce şifrelenmiş değer gelmeli.
	AddVariable(ctx context.Context, v *domain.EnvVariable, encryptedValue string) error

	// UpdateVariable, mevcut bir değişkeni günceller.
	UpdateVariable(ctx context.Context, varID string, encryptedValue string, v *domain.EnvVariable) error

	// DeleteVariable, belirtilen değişkeni siler.
	DeleteVariable(ctx context.Context, varID string) error

	// GetVariables, belirtilen ortama ait tüm değişkenleri döndürür.
	// UYARI: Value alanı şifreli (encrypted) değeri taşır; usecase katmanı çözmelidir.
	GetVariables(ctx context.Context, envID string) ([]domain.EnvVariable, error)
}
