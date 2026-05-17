package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "dockit-desktop"
	keyringUser    = "env-encryption-key"
)

// CryptoService, ortam değişkenleri için AES-256-GCM şifreleme/çözme işlemlerini yönetir.
// Şifreleme anahtarı uygulama başlangıcında işletim sisteminin güvenli keyring'ine
// kaydedilir; dosya sistemine yazılmaz.
type CryptoService struct {
	key []byte
}

// NewCryptoService, sistem keyring'den AES-256 anahtarı yükler.
// Eğer anahtar daha önce oluşturulmamışsa rastgele yeni bir anahtar oluşturur
// ve keyring'e kaydeder.
func NewCryptoService() (*CryptoService, error) {
	key, err := loadOrCreateKey()
	if err != nil {
		return nil, fmt.Errorf("crypto: anahtar başlatma hatası: %w", err)
	}
	return &CryptoService{key: key}, nil
}

// Encrypt, düz metin değeri AES-256-GCM ile şifreler.
// Çıktı, nonce + ciphertext birleşiminin base64 kodlanmış halidir.
// Her şifrelemede rastgele nonce kullanılır; aynı düz metin farklı çıktılar üretir.
func (c *CryptoService) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("crypto: cipher oluşturulamadı: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: GCM oluşturulamadı: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("crypto: nonce üretilemedi: %w", err)
	}

	// Seal: nonce'u başa ekleyip, sonrasına şifreli veriyi yaz
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt, Encrypt tarafından üretilen base64 kodlanmış şifreli metni çözer.
// Bozuk veri veya yanlış anahtar durumunda hata döner (kimlik doğrulama dahil).
func (c *CryptoService) Decrypt(encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("crypto: base64 çözme hatası: %w", err)
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("crypto: cipher oluşturulamadı: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: GCM oluşturulamadı: %w", err)
	}

	if len(data) < gcm.NonceSize() {
		return "", errors.New("crypto: şifreli veri çok kısa")
	}

	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("crypto: şifre çözme başarısız (bozuk veri veya yanlış anahtar): %w", err)
	}

	return string(plaintext), nil
}

// loadOrCreateKey, sistem keyring'den AES-256 anahtarını yükler.
// Eğer anahtar bulunamazsa kriptografik olarak güvenli rastgele bir anahtar
// oluşturur ve keyring'e kaydeder.
func loadOrCreateKey() ([]byte, error) {
	// Önce mevcut anahtarı bulmayı dene
	encoded, err := keyring.Get(keyringService, keyringUser)
	if err == nil {
		key, decodeErr := base64.StdEncoding.DecodeString(encoded)
		if decodeErr != nil {
			return nil, fmt.Errorf("crypto: keyring'deki anahtar geçersiz: %w", decodeErr)
		}
		if len(key) != 32 {
			return nil, errors.New("crypto: keyring'deki anahtar 32 byte olmalı")
		}
		return key, nil
	}

	// Anahtar bulunamadı → yeni AES-256 anahtarı oluştur
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("crypto: yeni anahtar üretilemedi: %w", err)
	}

	encoded = base64.StdEncoding.EncodeToString(key)
	if setErr := keyring.Set(keyringService, keyringUser, encoded); setErr != nil {
		return nil, fmt.Errorf("crypto: anahtar keyring'e kaydedilemedi: %w", setErr)
	}

	return key, nil
}
