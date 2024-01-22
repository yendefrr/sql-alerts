package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Config struct {
	Database             DatabaseConfig `json:"database"`
	Queries              []QueryConfig  `json:"queries"`
	BaseNotificationUrl  string         `json:"baseNotificationUrl"`
	NotificationMessage  string         `json:"notificationMessage"`
	CheckIntervalMinutes time.Duration  `json:"checkIntervalMinutes"`
}

type DatabaseConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	Name     string `json:"name"`
}

type QueryConfig struct {
	Name            string `json:"name"`
	Query           string `json:"query"`
	NotificationUrl string `json:"notificationUrl,omitempty"`
	Disabled        bool   `json:"disabled,omitempty"`
}

func main() {
	config, err := loadConfig("config.json")
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", config.Database.Username, config.Database.Password, config.Database.Host, config.Database.Port, config.Database.Name))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Println("Connection with database established")

	payload := strings.NewReader("Monitoring server started")

	resp, err := http.Post(config.BaseNotificationUrl, "text/plain", payload)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("HTTP request failed with status code: %d", resp.StatusCode)
	}

	log.Print("Initial notification sent")

	// Run monitoring loop
	for {
		for _, queryConfig := range config.Queries {
			if queryConfig.Disabled != true {
				err := monitorAndNotify(db, config, queryConfig)
				if err != nil {
					log.Printf("Error during monitoring: %v", err)
				}
			}

		}

		// Sleep for the specified interval before checking again
		time.Sleep(config.CheckIntervalMinutes * time.Minute)
	}
}

func monitorAndNotify(db *sql.DB, config Config, queryConfig QueryConfig) error {
	// Read processed IDs from the file
	processedIDs, err := readProcessedIDs(queryConfig.Name)
	if err != nil {
		return err
	}

	// Execute the SQL query
	rows, err := db.Query(queryConfig.Query)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Check for new rows and send notifications
	var newRows []int
	for rows.Next() {
		var id int

		err := rows.Scan(&id)
		if err != nil {
			return err
		}

		// Check if the row ID is not in the processedIDs list
		if !contains(processedIDs, id) {
			newRows = append(newRows, id)
		}
	}

	if len(newRows) > 0 {
		// Use query-specific NotificationUrl if provided, otherwise use the main one
		var url string
		if queryConfig.NotificationUrl != "" {
			url = queryConfig.NotificationUrl
		} else {
			url = config.BaseNotificationUrl
		}

		// Send HTTP POST request to the specified endpoint
		message := fmt.Sprintf(queryConfig.Name+": "+config.NotificationMessage, len(newRows))
		payload := strings.NewReader(message)

		resp, err := http.Post(url, "text/plain", payload)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP request failed with status code: %d", resp.StatusCode)
		}

		log.Printf("Notification sent for query %s: %s", queryConfig.Name, message)

		// Append new row IDs to the processedIDs list and update the file
		processedIDs = append(processedIDs, newRows...)
		err = writeProcessedIDs(queryConfig.Name, processedIDs)
		if err != nil {
			return err
		}
	}

	return nil
}

func loadConfig(filename string) (Config, error) {
	var config Config
	configFile, err := os.ReadFile(filename)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(configFile, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}

func readProcessedIDs(queryName string) ([]int, error) {
	filePath := fmt.Sprintf("./processed/%s_%s", queryName, "processed_ids.txt")

	// Check if the file exists, and create it if not
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if _, err := os.Create(filePath); err != nil {
			return nil, err
		}
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var processedIDs []int
	ids := strings.Split(string(content), "\n")
	for _, idStr := range ids {
		if idStr != "" {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return nil, err
			}
			processedIDs = append(processedIDs, id)
		}
	}

	return processedIDs, nil
}

func writeProcessedIDs(queryName string, processedIDs []int) error {
	filePath := fmt.Sprintf("./processed/%s_%s", queryName, "processed_ids.txt")
	idStr := ""
	for _, id := range processedIDs {
		idStr += fmt.Sprint(id) + "\n"
	}

	return os.WriteFile(filePath, []byte(idStr), 0644)
}

func contains(slice []int, item int) bool {
	for _, i := range slice {
		if i == item {
			return true
		}
	}
	return false
}
