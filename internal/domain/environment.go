package domain

import "time"

// Environment, API ortamı değişkenlerini gruplandıran bir ortamı temsil eder.
// Örneğin: "Development", "Staging", "Production"
type Environment struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	IsActive  bool          `json:"is_active"`
	Variables []EnvVariable `json:"variables"`
	CreatedAt time.Time     `json:"created_at"`
}

// EnvVariable, bir ortama ait tek bir değişkeni temsil eder.
// Tüm değerler AES-GCM ile şifreli saklanır; IsSecret=true olanlar
// frontend'e asla düz metin olarak gönderilmez.
type EnvVariable struct {
	ID          string `json:"id"`
	EnvID       string `json:"env_id"`
	Key         string `json:"key"`
	Value       string `json:"value"`       // Frontend'de: secret ise "••••••••", değilse düz metin
	IsSecret    bool   `json:"is_secret"`
	Description string `json:"description"`
}
