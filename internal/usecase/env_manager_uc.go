package usecase

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/google/uuid"

	"dockit-desktop/internal/domain"
	"dockit-desktop/internal/infrastructure/crypto"
	"dockit-desktop/internal/ports"
)

// secretMask, frontend'e gönderilen gizli değişkenlerin değerini maskeler.
const secretMask = "••••••••"

// varPattern, {{variable_name}} sözdizimini eşleştiren regex.
var varPattern = regexp.MustCompile(`\{\{([a-zA-Z0-9_]+)\}\}`)

// EnvManagerUsecase, API ortamlarını ve değişkenlerini yönetir.
// Şifreleme sorumluluğunu CryptoService'e, depolama sorumluluğunu EnvStorePort'a devreder.
type EnvManagerUsecase struct {
	mu      sync.RWMutex
	store   ports.EnvStorePort
	crypto  *crypto.CryptoService
}

// NewEnvManagerUsecase, bağımlılıkları enjekte ederek yeni bir usecase oluşturur.
func NewEnvManagerUsecase(store ports.EnvStorePort, cs *crypto.CryptoService) *EnvManagerUsecase {
	return &EnvManagerUsecase{
		store:  store,
		crypto: cs,
	}
}

// =============================================================================
// Ortam Yönetimi
// =============================================================================

// CreateEnvironment, yeni bir ortam oluşturur.
func (uc *EnvManagerUsecase) CreateEnvironment(ctx context.Context, name string) (*domain.Environment, error) {
	if name == "" {
		return nil, fmt.Errorf("env: ortam adı boş olamaz")
	}
	env := &domain.Environment{
		ID:        uuid.NewString(),
		Name:      name,
		IsActive:  false,
		Variables: nil,
		CreatedAt: time.Now().UTC(),
	}
	if err := uc.store.CreateEnvironment(ctx, env); err != nil {
		return nil, fmt.Errorf("env: ortam oluşturulamadı: %w", err)
	}
	return env, nil
}

// ListEnvironments, tüm ortamları değişkenleriyle birlikte döndürür.
// GÜVENLİK: IsSecret=true olan değişkenlerin değerleri "••••••••" olarak maskelenir.
// IsSecret=false olan değişkenler çözülerek (decrypted) döndürülür.
func (uc *EnvManagerUsecase) ListEnvironments(ctx context.Context) ([]domain.Environment, error) {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	envs, err := uc.store.ListEnvironments(ctx)
	if err != nil {
		return nil, err
	}

	for i := range envs {
		vars, err := uc.store.GetVariables(ctx, envs[i].ID)
		if err != nil {
			return nil, err
		}
		for j := range vars {
			if vars[j].IsSecret {
				// Gizli değişkenler asla frontend'e gönderilmez
				vars[j].Value = secretMask
			} else {
				// Şifreyi çöz ve düz metin gönder
				plain, decErr := uc.crypto.Decrypt(vars[j].Value)
				if decErr != nil {
					vars[j].Value = "[şifre çözme hatası]"
				} else {
					vars[j].Value = plain
				}
			}
		}
		envs[i].Variables = vars
	}
	return envs, nil
}

// DeleteEnvironment, belirtilen ortamı tüm değişkenleriyle siler.
func (uc *EnvManagerUsecase) DeleteEnvironment(ctx context.Context, id string) error {
	return uc.store.DeleteEnvironment(ctx, id)
}

// SetActiveEnvironment, belirtilen ortamı aktif yapar.
func (uc *EnvManagerUsecase) SetActiveEnvironment(ctx context.Context, id string) error {
	return uc.store.SetActiveEnvironment(ctx, id)
}

// =============================================================================
// Değişken Yönetimi
// =============================================================================

// AddVariable, aktif veya belirtilen ortama yeni bir değişken ekler.
// value parametresi düz metin olarak alınır; bu fonksiyon şifreleyerek saklar.
func (uc *EnvManagerUsecase) AddVariable(ctx context.Context, envID, key, value, description string, isSecret bool) (*domain.EnvVariable, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	if key == "" {
		return nil, fmt.Errorf("env: değişken anahtarı boş olamaz")
	}

	// Tüm değerleri şifrele (secret olsun ya da olmasın — rest-at-encryption)
	encrypted, err := uc.crypto.Encrypt(value)
	if err != nil {
		return nil, fmt.Errorf("env: değer şifrelenemedi: %w", err)
	}

	v := &domain.EnvVariable{
		ID:          uuid.NewString(),
		EnvID:       envID,
		Key:         key,
		IsSecret:    isSecret,
		Description: description,
	}

	if err := uc.store.AddVariable(ctx, v, encrypted); err != nil {
		return nil, fmt.Errorf("env: değişken kaydedilemedi: %w", err)
	}

	// Frontend'e güvenli sürümü döndür
	if isSecret {
		v.Value = secretMask
	} else {
		v.Value = value
	}
	return v, nil
}

// UpdateVariable, mevcut bir değişkeni günceller.
// Eğer value boşsa mevcut değer korunur (parola güncelleme UX için).
func (uc *EnvManagerUsecase) UpdateVariable(ctx context.Context, varID, key, value, description string, isSecret bool) error {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	encrypted, err := uc.crypto.Encrypt(value)
	if err != nil {
		return fmt.Errorf("env: değer şifrelenemedi: %w", err)
	}

	v := &domain.EnvVariable{
		Key:         key,
		IsSecret:    isSecret,
		Description: description,
	}
	return uc.store.UpdateVariable(ctx, varID, encrypted, v)
}

// DeleteVariable, belirtilen değişkeni siler.
func (uc *EnvManagerUsecase) DeleteVariable(ctx context.Context, varID string) error {
	return uc.store.DeleteVariable(ctx, varID)
}

// =============================================================================
// Template Resolution (sunucu taraflı interpolasyon)
// =============================================================================

// ResolveTemplate, verilen şablondaki {{key}} yer tutucularını aktif ortamın
// değişken değerleriyle değiştirir. Bu işlem tamamen Go backend'de gerçekleşir;
// ham değerler (özellikle secret'lar) hiçbir zaman frontend'e gönderilmez.
//
// Eşleşmeyen yer tutucular olduğu gibi bırakılır (hata döndürülmez).
func (uc *EnvManagerUsecase) ResolveTemplate(ctx context.Context, template string) (string, error) {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	// {{...}} içermeyen şablonları hızlı döndür
	if !varPattern.MatchString(template) {
		return template, nil
	}

	// Aktif ortamı bul
	activeID, err := uc.store.GetActiveEnvironmentID(ctx)
	if err != nil {
		return template, fmt.Errorf("env: aktif ortam alınamadı: %w", err)
	}
	if activeID == "" {
		// Aktif ortam yok; şablon olduğu gibi döner
		return template, nil
	}

	// Aktif ortamın değişkenlerini çek ve şifreleri çöz
	encVars, err := uc.store.GetVariables(ctx, activeID)
	if err != nil {
		return template, fmt.Errorf("env: değişkenler alınamadı: %w", err)
	}

	// key → plain_value haritası oluştur
	resolved := make(map[string]string, len(encVars))
	for _, v := range encVars {
		plain, decErr := uc.crypto.Decrypt(v.Value)
		if decErr != nil {
			// Şifre çözme başarısız → yer tutucuyu koru
			continue
		}
		resolved[v.Key] = plain
	}

	// Regex ile {{key}} → değer değişimi
	result := varPattern.ReplaceAllStringFunc(template, func(match string) string {
		// "{{key}}" → "key"
		key := match[2 : len(match)-2]
		if val, ok := resolved[key]; ok {
			return val
		}
		return match // Eşleşme bulunamazsa orijinal yer tutucu korunur
	})

	return result, nil
}
