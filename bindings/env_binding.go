package bindings

import (
	"context"
	"dockit-desktop/internal/domain"
	"dockit-desktop/internal/usecase"
)

// EnvBinding, frontend'in ortam yönetimi için çağırabileceği metodları sunar.
//
// GÜVENLİK NOTU:
//   - IsSecret=true olan değişkenler frontend'e ASLA düz metin olarak gönderilmez.
//   - {{variable}} interpolasyonu tamamen backend'de yapılır (ResolveTemplate).
//   - ExecuteRequestWithEnv, URL ve payload'ı çözümledikten sonra isteği gönderir;
//     çözümlenmiş değerler frontend'e dönmez, yalnızca HTTP yanıtı döner.
type EnvBinding struct {
	ctx   context.Context
	envUC *usecase.EnvManagerUsecase
	apiUC *usecase.APIUsecase
}

// NewEnvBinding, bağımlılıkları enjekte eder.
// apiUC, ortam değişkenleriyle interpolasyonlu istek göndermek için gereklidir.
func NewEnvBinding(envUC *usecase.EnvManagerUsecase, apiUC *usecase.APIUsecase) *EnvBinding {
	return &EnvBinding{envUC: envUC, apiUC: apiUC}
}

func (b *EnvBinding) Startup(ctx context.Context) {
	b.ctx = ctx
}

// =============================================================================
// Ortam Yönetimi
// =============================================================================

// CreateEnvironment, yeni bir ortam oluşturur.
func (b *EnvBinding) CreateEnvironment(name string) (*domain.Environment, error) {
	return b.envUC.CreateEnvironment(b.ctx, name)
}

// ListEnvironments, tüm ortamları değişkenleriyle birlikte döndürür.
// Secret değişkenler maskelenmiş ("••••••••") olarak gelir.
func (b *EnvBinding) ListEnvironments() ([]domain.Environment, error) {
	return b.envUC.ListEnvironments(b.ctx)
}

// DeleteEnvironment, belirtilen ortamı ve tüm değişkenlerini siler.
func (b *EnvBinding) DeleteEnvironment(id string) error {
	return b.envUC.DeleteEnvironment(b.ctx, id)
}

// SetActiveEnvironment, belirtilen ortamı aktif yapar.
func (b *EnvBinding) SetActiveEnvironment(id string) error {
	return b.envUC.SetActiveEnvironment(b.ctx, id)
}

// =============================================================================
// Değişken Yönetimi
// =============================================================================

// AddVariable, belirtilen ortama yeni bir değişken ekler.
// isSecret=true ise değer frontend'de maskelenir; her iki durumda da
// değer AES-GCM ile şifreli saklanır.
func (b *EnvBinding) AddVariable(envID, key, value, description string, isSecret bool) (*domain.EnvVariable, error) {
	return b.envUC.AddVariable(b.ctx, envID, key, value, description, isSecret)
}

// UpdateVariable, mevcut bir değişkeni günceller.
func (b *EnvBinding) UpdateVariable(varID, key, value, description string, isSecret bool) error {
	return b.envUC.UpdateVariable(b.ctx, varID, key, value, description, isSecret)
}

// DeleteVariable, belirtilen değişkeni siler.
func (b *EnvBinding) DeleteVariable(varID string) error {
	return b.envUC.DeleteVariable(b.ctx, varID)
}

// =============================================================================
// Ortam Değişkenli HTTP İsteği
// =============================================================================

// ExecuteRequestWithEnv, URL ve payload içindeki {{variable}} yer tutucularını
// aktif ortamın değişkenleriyle çözümleyerek HTTP isteği gönderir.
//
// GÜVENLİK: Çözümlenmiş (ham) değerler frontend'e dönmez; yalnızca HTTP yanıtı döner.
func (b *EnvBinding) ExecuteRequestWithEnv(method, rawURL, rawPayload string) (*domain.APIRequest, error) {
	// URL'yi çözümle
	resolvedURL, err := b.envUC.ResolveTemplate(b.ctx, rawURL)
	if err != nil {
		return nil, err
	}

	// Payload'ı çözümle
	resolvedPayload, err := b.envUC.ResolveTemplate(b.ctx, rawPayload)
	if err != nil {
		return nil, err
	}

	// Çözümlenmiş değerlerle isteği gönder ve DB'ye kaydet
	return b.apiUC.ExecuteAndSaveRequest(b.ctx, method, resolvedURL, resolvedPayload)
}
