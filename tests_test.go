package gormx

import (
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func init() {
	db := newDB()
	testDB(db)
}

func ptr[T any](v T) *T {
	return &v
}

func newDB() *gorm.DB {
	dbDSN := os.Getenv("GORM_DSN")
	if dbDSN == "" {
		dbDSN = "gorm:gorm@tcp(localhost:9910)/gorm?charset=utf8&parseTime=True&loc=Local"
	}
	db, err := gorm.Open(mysql.Open(dbDSN), &gorm.Config{})
	if err != nil {
		log.Printf("connect to mysql fail: %s\n", err)
		os.Exit(1)
	}

	if debug := os.Getenv("DEBUG"); debug == "true" {
		db.Logger = db.Logger.LogMode(logger.Info)
	} else if debug == "false" {
		db.Logger = db.Logger.LogMode(logger.Silent)
	}

	return db
}

func testDB(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("failed to connect database, got error %v", err)
		os.Exit(1)
	}

	err = sqlDB.Ping()
	if err != nil {
		log.Printf("failed to ping sqlDB, got error %v", err)
		os.Exit(1)
	}

	if db.Dialector.Name() == "sqlite" {
		db.Exec("PRAGMA foreign_keys = ON")
	}
}
