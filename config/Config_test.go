package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	testEnvFile string = ".testenv"
	testConfig  AppConfig
)

func checkErr(err error) {
	if err != nil {
		panic(fmt.Sprintf("could not execute test preparation. Error: %s", err))
	}
}

func writeTestEnv(fileName string) {
	f, err := os.Create(fileName)
	checkErr(err)
	defer f.Close()
	w := bufio.NewWriter(f)
	_, err = w.WriteString("GIN_MODE=\"debug\"\n")
	checkErr(err)
	_, err = w.WriteString("SERVER_HOST=\"127.0.0.1\"\n")
	checkErr(err)
	_, err = w.WriteString("SERVER_PORT=\"9999\"\n")
	checkErr(err)
	w.Flush()
}

func deleteEnvFile(fileName string) {
	err := os.Remove(fileName)
	checkErr(err)
}

func unsetEnvVars() {
	os.Unsetenv("GIN_MODE")
	os.Unsetenv("SERVER_HOST")
	os.Unsetenv("SERVER_PORT")
}

func TestLoadConfigNoEnvFileReturnsError(t *testing.T) {
	err := loadConfig("file_does_not_exist.txt")
	assert.NotNil(t, err)
	fmt.Printf("error: %v", err)

	assert.True(t, os.IsNotExist(err))
}

func TestLoadConfigWithEnvFileReturnsNoError(t *testing.T) {
	writeTestEnv(testEnvFile)
	defer deleteEnvFile(testEnvFile)
	err := loadConfig(testEnvFile)
	defer unsetEnvVars()

	assert.Nil(t, err)
	assert.EqualValues(t, "127.0.0.1", os.Getenv("SERVER_HOST"))
	assert.EqualValues(t, "debug", os.Getenv("GIN_MODE"))
}

func TestInitConfigWithEnvFileSetsValues(t *testing.T) {
	writeTestEnv(testEnvFile)
	defer deleteEnvFile(testEnvFile)
	err := InitConfig(testEnvFile, &testConfig)

	assert.Nil(t, err)
	assert.EqualValues(t, 10, testConfig.Server.GracefulShutdownTime)
	assert.EqualValues(t, "debug", testConfig.Gin.Mode)
}

func TestCheckFilePathEmptyPathKeepsPathEmpty(t *testing.T) {
	testPath := ""
	checkFilePath(&testPath)
	assert.EqualValues(t, "", testPath)
}

func TestCheckFilePathCorrectPathReturnsCorrectPath(t *testing.T) {
	testPath := "C:\\TEMP"
	checkFilePath(&testPath)
	assert.EqualValues(t, "C:\\temp", testPath)
}

func TestCheckFilePathWeirdPathReturnsCorrectPath(t *testing.T) {
	testPath := "C:\\TEMP\\..\\..\\..\\etc"
	checkFilePath(&testPath)
	assert.EqualValues(t, "C:\\etc", testPath)
}

func TestValidateConfigInvalidExportMinuteReturnsError(t *testing.T) {
	var cfg AppConfig
	cfg.Server.GracefulShutdownTime = 10
	cfg.Crawl.CrawlCycleMin = 10
	cfg.Export.ExportMinute = 60
	cfg.Export.StatusQueryCycleSec = 5

	err := validateConfig(&cfg)

	assert.NotNil(t, err)
	assert.EqualValues(t, "export minute must be between 0 and 59", err.Error())
}

func TestValidateConfigInvalidCrawlCycleReturnsError(t *testing.T) {
	var cfg AppConfig
	cfg.Server.GracefulShutdownTime = 10
	cfg.Crawl.CrawlCycleMin = 0
	cfg.Export.ExportMinute = 59
	cfg.Export.StatusQueryCycleSec = 5

	err := validateConfig(&cfg)

	assert.NotNil(t, err)
	assert.EqualValues(t, "crawl cycle must be greater than 0", err.Error())
}

func TestValidateConfigInvalidStatusQueryCycleReturnsError(t *testing.T) {
	var cfg AppConfig
	cfg.Server.GracefulShutdownTime = 10
	cfg.Crawl.CrawlCycleMin = 10
	cfg.Export.ExportMinute = 59
	cfg.Export.StatusQueryCycleSec = 0

	err := validateConfig(&cfg)

	assert.NotNil(t, err)
	assert.EqualValues(t, "status query cycle must be greater than 0", err.Error())
}

func TestValidateConfigInvalidRootFolderReturnsError(t *testing.T) {
	var cfg AppConfig
	cfg.Server.GracefulShutdownTime = 10
	cfg.Crawl.CrawlCycleMin = 10
	cfg.Export.ExportMinute = 59
	cfg.Export.StatusQueryCycleSec = 5
	cfg.Crawl.RootFolder = filepath.Join(t.TempDir(), "missing")

	err := validateConfig(&cfg)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "root folder is not accessible")
}

func TestValidateConfigRootFolderFileReturnsError(t *testing.T) {
	var cfg AppConfig
	cfg.Server.GracefulShutdownTime = 10
	cfg.Crawl.CrawlCycleMin = 10
	cfg.Export.ExportMinute = 59
	cfg.Export.StatusQueryCycleSec = 5
	rootFile := filepath.Join(t.TempDir(), "root-file")
	os.WriteFile(rootFile, []byte("test"), 0644)
	cfg.Crawl.RootFolder = rootFile

	err := validateConfig(&cfg)

	assert.NotNil(t, err)
	assert.EqualValues(t, "root folder must be a directory", err.Error())
}

func TestValidateConfigTlsMissingCertReturnsError(t *testing.T) {
	var cfg AppConfig
	cfg.Server.GracefulShutdownTime = 10
	cfg.Crawl.CrawlCycleMin = 10
	cfg.Export.ExportMinute = 59
	cfg.Export.StatusQueryCycleSec = 5
	cfg.Server.UseTls = true
	cfg.Server.CertFile = filepath.Join(t.TempDir(), "missing-cert.pem")
	cfg.Server.KeyFile = filepath.Join(t.TempDir(), "missing-key.pem")

	err := validateConfig(&cfg)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "TLS certificate file is not accessible")
}

func TestValidateConfigTlsMissingKeyReturnsError(t *testing.T) {
	var cfg AppConfig
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	os.WriteFile(certFile, []byte("cert"), 0644)
	cfg.Server.GracefulShutdownTime = 10
	cfg.Crawl.CrawlCycleMin = 10
	cfg.Export.ExportMinute = 59
	cfg.Export.StatusQueryCycleSec = 5
	cfg.Server.UseTls = true
	cfg.Server.CertFile = certFile
	cfg.Server.KeyFile = filepath.Join(dir, "missing-key.pem")

	err := validateConfig(&cfg)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "TLS key file is not accessible")
}

func TestValidateConfigValidRootFolderReturnsNoError(t *testing.T) {
	var cfg AppConfig
	cfg.Server.GracefulShutdownTime = 10
	cfg.Crawl.CrawlCycleMin = 10
	cfg.Export.ExportMinute = 59
	cfg.Export.StatusQueryCycleSec = 5
	cfg.Crawl.RootFolder = t.TempDir()

	err := validateConfig(&cfg)

	assert.Nil(t, err)
}
