package service

import (
	"testing"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
)

var (
	cfg           config.AppConfig
	fileRepo      repositories.DefaultFileRepository
	exportService DefaultExportService
)

func setupTest(t *testing.T) func() {
	config.InitConfig(config.EnvFile, &cfg)
	fileRepo = repositories.NewFileRepository(&cfg)
	exportService = NewExportService(&cfg, &fileRepo)
	return func() {
	}
}
