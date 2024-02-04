package service

import (
	"testing"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/stretchr/testify/assert"
)

var (
	cfgCrawl config.AppConfig
	crawlSvc DefaultCrawlService
)

func setupTestCrawl(t *testing.T) func() {
	config.InitConfig(config.EnvFile, &cfgCrawl)
	crawlSvc = NewCrawlService(&cfgCrawl, nil)
	return func() {
	}
}

func Test_parseEventId_WithNumId_ReturnsId(t *testing.T) {
	teardown := setupTestCrawl(t)
	defer teardown()
	id := crawlSvc.parseEventId("Z:\\sendungen\\2024\\02\\04\\18-00\\2024-02-03_11-31-57-id34067-seniorenradio-CRKaleidoskop2024.mp3")
	assert.EqualValues(t, 34067, id)
}

func Test_parseEventId_NoId_ReturnsZero(t *testing.T) {
	teardown := setupTestCrawl(t)
	defer teardown()
	id := crawlSvc.parseEventId("Z:\\sendungen\\2024\\02\\04\\18-00\\2024-02-03_11-31-57-seniorenradio-CRKaleidoskop2024.mp3")
	assert.EqualValues(t, 0, id)
}

func Test_parseEventId_NoNumId_ReturnsZero(t *testing.T) {
	teardown := setupTestCrawl(t)
	defer teardown()
	id := crawlSvc.parseEventId("Z:\\sendungen\\2024\\02\\04\\18-00\\2024-02-03_11-31-57-id34AB067-seniorenradio-CRKaleidoskop2024.mp3")
	assert.EqualValues(t, 0, id)
}
