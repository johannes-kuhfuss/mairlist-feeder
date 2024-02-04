package service

import (
	"testing"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
)

var (
	cfgA          config.AppConfig
	fileRepoA     repositories.DefaultFileRepository
	calCmsService DefaultCalCmsService
)

func setupTestA(t *testing.T) func() {
	config.InitConfig(config.EnvFile, &cfgA)
	fileRepoA = repositories.NewFileRepository(&cfgA)
	calCmsService = NewCalCmsService(&cfgA, &fileRepo)
	return func() {
	}
}

func Test__ReturnsData(t *testing.T) {
	teardown := setupTestA(t)
	defer teardown()
}
