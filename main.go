package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"dockit-desktop/bindings"
	"dockit-desktop/internal/infrastructure/crypto"
	"dockit-desktop/internal/infrastructure/database"
	"dockit-desktop/internal/infrastructure/docker"
	"dockit-desktop/internal/infrastructure/httpclient"
	"dockit-desktop/internal/usecase"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// 1. Initialize Infrastructure (Ports)
	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		log.Fatalf("Failed to initialize Docker client: %v", err)
	}

	dbClient, err := database.NewSQLiteDB("dockit.db")
	if err != nil {
		log.Fatalf("Failed to initialize SQLite database: %v", err)
	}

	httpClient := httpclient.NewHTTPClient()

	// 2. Initialize Usecases
	dockerUC := usecase.NewDockerUsecase(dockerClient)
	dbUC := usecase.NewDatabaseUsecase(dbClient)
	apiUC := usecase.NewAPIUsecase(httpClient, dbClient)
	dbManagerUC := usecase.NewDbManagerUsecase()

	// Ortam yönetimi için kripto servisi (keyring tabanlı AES-256-GCM)
	cryptoService, err := crypto.NewCryptoService()
	if err != nil {
		log.Fatalf("Kripto servisi başlatılamadı: %v", err)
	}
	envUC := usecase.NewEnvManagerUsecase(dbClient, cryptoService)

	// 3. Initialize Bindings
	dockerBinding := bindings.NewDockerBinding(dockerUC)
	dbBinding := bindings.NewDatabaseBinding(dbUC)
	apiBinding := bindings.NewAPIBinding(apiUC)
	dbManagerBinding := bindings.NewDbManagerBinding(dbManagerUC)
	envBinding := bindings.NewEnvBinding(envUC, apiUC)

	// Create an instance of the app structure
	app := NewApp(dockerBinding, dbBinding, apiBinding, dbManagerBinding, envBinding)

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "dockit-desktop",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
			dockerBinding,
			dbBinding,
			apiBinding,
			dbManagerBinding,
			envBinding,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
