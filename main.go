package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	defaultConfigFileName = "config.json"
	defaultConfigDir      = ".config/sqlal"
)

var (
	configFile string
	version    bool
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
	// Automatically create default config and processed directories if they don't exist
	if err := initializeDirectories(); err != nil {
		log.Fatalf("Failed to initialize directories: %v", err)
	}

	flag.StringVar(&configFile, "config", getDefaultConfigFilePath(), "Path to configuration file")
	flag.BoolVar(&version, "version", false, "Print version information and exit")
	flag.Parse()

	if version {
		fmt.Println("0.4.3")
		return
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "start":
			start()
			return
		case "stop":
			stop()
			return
		}
	}

	config, err := loadConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}

	// Create processed directory if it doesn't exist
	processedDir := filepath.Join(getUserConfigDir(), "processed")
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

// Helper function to initialize default directories and config file
func initializeDirectories() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(homeDir, defaultConfigDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	processedDir := filepath.Join(configDir, "processed")
	if err := os.MkdirAll(processedDir, 0755); err != nil {
		return err
	}

	configFilePath := filepath.Join(configDir, defaultConfigFileName)
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		// Config file doesn't exist, create default config
		defaultConfig := Config{
			BaseNotificationUrl:  "https://ntfy.sh/base",
			NotificationMessage:  "New %d rows",
			CheckIntervalMinutes: 1,
		}
		if err := writeConfig(defaultConfig, configFilePath); err != nil {
			return err
		}
	}

	return nil
}

// Helper function to write default config to file
func writeConfig(config Config, filename string) error {
	configData, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, configData, 0644)
}

// Helper function to get the default configuration file path
func getDefaultConfigFilePath() string {
	return filepath.Join(getUserConfigDir(), defaultConfigFileName)
}

// Helper function to get the user's configuration directory
func getUserConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get user's home directory: %v", err)
	}
	return filepath.Join(homeDir, defaultConfigDir)
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

func start() {
	fmt.Println("Starting sqlal...")
	cmd := exec.Command(os.Args[0])
	err := cmd.Start()
	if err != nil {
		log.Fatalf("Failed to start sqlal: %v", err)
	}
	fmt.Println("sqlal started successfully.")
}

func stop() {
	fmt.Println("Stopping sqlal...")
	out, err := exec.Command("pkill", "-f", os.Args[0]).CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to stop sqlal: %v", err)
	}
	fmt.Println(string(out))
	fmt.Println("sqlal stopped.")
}
