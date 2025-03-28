package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

var SDB *sql.DB

func InitMySQL() error {
	dsn := fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/journey_app?charset=utf8mb4&parseTime=True&loc=Local", os.Getenv("DBU"), os.Getenv("DBP"))
	var err error
	SDB, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("sql.Open error: %w", err)
	}

	if err := SDB.Ping(); err != nil {
		return fmt.Errorf("db.Ping error: %w", err)
	}

	if err := createTables(); err != nil {
		fmt.Println("createTables error occurred: ", err)
		return err
	}

	fmt.Println("MySQL connected and schema ensured.")
	return nil
}

func createTables() error {
	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		user_id VARCHAR(36) PRIMARY KEY,
		username VARCHAR(50) NOT NULL UNIQUE,
		password VARCHAR(255) NOT NULL,
		salt VARCHAR(50) NOT NULL,
		api_key VARCHAR(100) NOT NULL UNIQUE,
		api_key_created DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		api_key_last_used DATETIME,
		api_key_expires_at DATETIME,
		font VARCHAR(50),
		INDEX idx_username (username)
	);`

	entriesTable := `
	CREATE TABLE IF NOT EXISTS entries (
		entry_id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL,
		username VARCHAR(50) NOT NULL,
		text TEXT NOT NULL,
		timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_updated DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
		INDEX idx_user_timestamp (user_id, timestamp),
		INDEX idx_username (username)
	);`

	entryLocationsTable := `
	CREATE TABLE IF NOT EXISTS entry_locations (
		location_id BIGINT AUTO_INCREMENT PRIMARY KEY,
		entry_id VARCHAR(36) NOT NULL,
		latitude DOUBLE NOT NULL,
		longitude DOUBLE NOT NULL,
		display_name VARCHAR(255),
		FOREIGN KEY (entry_id) REFERENCES entries(entry_id) ON DELETE CASCADE,
		INDEX idx_entry_id (entry_id)
	);`

	entryTagsTable := `
	CREATE TABLE IF NOT EXISTS entry_tags (
		tag_id BIGINT AUTO_INCREMENT PRIMARY KEY,
		entry_id VARCHAR(36) NOT NULL,
		tag_key VARCHAR(50) NOT NULL,
		tag_value VARCHAR(255),
		FOREIGN KEY (entry_id) REFERENCES entries(entry_id) ON DELETE CASCADE,
		INDEX idx_entry_tag (entry_id, tag_key),
		INDEX idx_tag_key (tag_key)
	);`

	entryImagesTable := `
	CREATE TABLE IF NOT EXISTS entry_images (
		image_id BIGINT AUTO_INCREMENT PRIMARY KEY,
		entry_id VARCHAR(36) NOT NULL,
		image_url VARCHAR(255) NOT NULL,
		FOREIGN KEY (entry_id) REFERENCES entries(entry_id) ON DELETE CASCADE,
		INDEX idx_entry_id (entry_id)
	);`

	// Analytics
	analyticsEventsTable := `
	CREATE TABLE IF NOT EXISTS analytics_events (
		event_id BIGINT AUTO_INCREMENT PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL,
		event_type VARCHAR(100) NOT NULL,
		object_type VARCHAR(100),
		object_id VARCHAR(36),
		event_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		meta_data JSON,
		FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
		INDEX idx_user_event_time (user_id, event_time),
		INDEX idx_event_type (event_type)
	);`

	if _, err := SDB.Exec(usersTable); err != nil {
		return err
	}
	if _, err := SDB.Exec(entriesTable); err != nil {
		return err
	}
	if _, err := SDB.Exec(entryLocationsTable); err != nil {
		return err
	}
	if _, err := SDB.Exec(entryTagsTable); err != nil {
		return err
	}
	if _, err := SDB.Exec(entryImagesTable); err != nil {
		return err
	}
	if _, err := SDB.Exec(analyticsEventsTable); err != nil {
		return err
	}

	return nil
}
