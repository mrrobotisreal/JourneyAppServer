package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type LogManager struct {
	Logger *log.Logger
	File   *os.File
	Mutex  sync.Mutex
}

var LM = &LogManager{}

func SetupDailyLogging() error {
	logDir := "./logs"

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	openLogFile := func() (*os.File, error) {
		dateStr := time.Now().Format("2006_01_02")
		logFile := filepath.Join(logDir, dateStr+".log")
		return os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	}

	initialFile, err := openLogFile()
	if err != nil {
		return fmt.Errorf("failed to open initial log file: %v", err)
	}
	LM.Mutex.Lock()
	LM.File = initialFile
	LM.Logger = log.New(initialFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	LM.Mutex.Unlock()

	LM.Logger.Println("Daily logging initialized successfully")

	go func() {
		for {
			now := time.Now()
			nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
			durationUntilMidnight := nextMidnight.Sub(now)

			time.Sleep(durationUntilMidnight)

			LM.Mutex.Lock()
			newFile, err := openLogFile()
			if err != nil {
				log.Printf("Failed to rotate log file: %v", err)
			} else {
				var oldFile *os.File
				oldFile = LM.File
				if oldFile != nil {
					if closeErr := oldFile.Close(); closeErr != nil {
						log.Printf("Error closing old log file: %v", closeErr)
					}
				}
				LM.File = newFile
				LM.Logger = log.New(newFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
				LM.Logger.Println("Log file rotated successfully")
			}
			LM.Mutex.Unlock()
		}
	}()

	return nil
}
