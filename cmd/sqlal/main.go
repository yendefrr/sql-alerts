package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	configFile := flag.String("config", "config.json", "Path to configuration file")
	flag.Parse()

	config, err := loadConfig(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	// Create processed directory if it doesn't exist
	processedDir := "/var/lib/sqlal/processed"
	if err := os.MkdirAll(processedDir, 0755); err != nil {
		log.Fatalf("Failed to create processed directory: %v", err)
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
			if !queryConfig.Disabled {
				err := monitorAndNotify(db, config, queryConfig, processedDir)
				if err != nil {
					log.Printf("Error during monitoring: %v", err)
				}
			}
		}

		// Sleep for the specified interval before checking again
		time.Sleep(config.CheckIntervalMinutes * time.Minute)
	}
}

func monitorAndNotify(db *sql.DB, config Config, queryConfig QueryConfig, processedDir string) error {
    // Read processed IDs from the file
    processedIDs, err := readProcessedIDs(queryConfig.Name, processedDir)
    if err != nil {
        return err
    }

    // Get new rows from the database
    newRows, err := getNewRows(db, queryConfig.Query, processedIDs)
    if err != nil {
        return err
    }

    // Send notifications for new rows
    if len(newRows) > 0 {
        err := sendNotifications(config, queryConfig, newRows)
        if err != nil {
            return err
        }

        // Update processedIDs with new rows
        processedIDs = append(processedIDs, newRows...)
        err = writeProcessedIDs(queryConfig.Name, processedIDs, processedDir)
        if err != nil {
            return err
        }
    }

    return nil
}

func getNewRows(db *sql.DB, query string, processedIDs []int) ([]int, error) {
    rows, err := db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var newRows []int
    for rows.Next() {
        var id int
        err := rows.Scan(&id)
        if err != nil {
            return nil, err
        }
        if !contains(processedIDs, id) {
            newRows = append(newRows, id)
        }
    }
    return newRows, nil
}

func sendNotifications(config Config, queryConfig QueryConfig, newRows []int) error {
    var url string
    if queryConfig.NotificationUrl != "" {
        url = queryConfig.NotificationUrl
    } else {
        url = config.BaseNotificationUrl
    }

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

func readProcessedIDs(queryName, directory string) ([]int, error) {
	filePath := filepath.Join(directory, fmt.Sprintf("%s_%s", queryName, "processed_ids.txt"))

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
func writeProcessedIDs(queryName string, processedIDs []int, directory string) error {
	filePath := filepath.Join(directory, fmt.Sprintf("%s_%s", queryName, "processed_ids.txt"))
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
