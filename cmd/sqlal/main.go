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

	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/go-sql-driver/mysql"

	"github.com/yendefrr/sql-alerts/internal"
)

var version = "0.4.8"

const (
	defaultConfigFileName = "config.json"
	defaultConfigDir      = ".config/sqlal"
)

var (
	configFile  string
	flagVersion bool
)

func main() {
	if err := initializeDirectories(); err != nil {
		log.Fatalf("Failed to initialize directories: %v", err)
	}

	flag.StringVar(&configFile, "config", getDefaultConfigFilePath(), "Path to configuration file")
	flag.BoolVar(&flagVersion, "v", false, "Print version information and exit")
	flag.Parse()

	if flagVersion {
		printVersion()
		return
	}

	homeDir, _ := os.UserHomeDir()
	configFilePath := filepath.Join(homeDir, defaultConfigDir, defaultConfigFileName)

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		config()
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
		case "restart":
			restart()
			return
		case "config":
			config()
			return
		}
	}

	config, err := loadConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}

	processedDir := createProcessedDir()

	db := connectToDatabase(config)
	defer db.Close()

	sendInitialNotification(config)

	runMonitoringLoop(db, config, processedDir)
}

func printVersion() {
	fmt.Println(version)
	os.Exit(0)
}

func createProcessedDir() string {
	processedDir := filepath.Join(getUserConfigDir(), "processed")
	if err := os.MkdirAll(processedDir, 0755); err != nil {
		log.Fatalf("Failed to create processed directory: %v", err)
	}
	return processedDir
}

func connectToDatabase(config internal.Config) *sql.DB {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", config.Database.Username, config.Database.Password, config.Database.Host, config.Database.Port, config.Database.Name))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connection with database established")
	return db
}

func sendInitialNotification(config internal.Config) {
	payload := strings.NewReader("Monitoring server started")
	resp, err := http.Post(config.BaseNotificationURL, "text/plain", payload)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("HTTP request failed with status code: %d", resp.StatusCode)
	}
	log.Print("Initial notification sent")
}

func runMonitoringLoop(db *sql.DB, config internal.Config, processedDir string) {
	for {
		for _, queryConfig := range config.Queries {
			if !queryConfig.Disabled {
				err := monitorAndNotify(db, config, queryConfig, processedDir)
				if err != nil {
					log.Printf("Error during monitoring: %v", err)
				}
			}
		}
		time.Sleep(time.Duration(config.CheckIntervalSeconds) * time.Second)
	}
}

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

	return nil
}

func getDefaultConfigFilePath() string {
	return filepath.Join(getUserConfigDir(), defaultConfigFileName)
}

func getUserConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get user's home directory: %v", err)
	}
	return filepath.Join(homeDir, defaultConfigDir)
}

func monitorAndNotify(db *sql.DB, config internal.Config, queryConfig internal.QueryConfig, processedDir string) error {
	processedIDs, err := readProcessedIDs(queryConfig.Name, processedDir)
	if err != nil {
		return err
	}

	newRows, err := getNewRows(db, queryConfig.Query, processedIDs)
	if err != nil {
		return err
	}

	if len(newRows) > 0 {
		err := sendNotifications(config, queryConfig, newRows)
		if err != nil {
			return err
		}

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

func sendNotifications(config internal.Config, queryConfig internal.QueryConfig, newRows []int) error {
	var url string
	if queryConfig.NotificationURL != "" {
		url = queryConfig.NotificationURL
	} else {
		url = config.BaseNotificationURL
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

func loadConfig(filename string) (internal.Config, error) {
	var config internal.Config
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
	cmd := exec.Command("pgrep", "-f", os.Args[0])

	output, _ := cmd.Output()
	if len(output) != 0 {
		fmt.Println("Already started!")
		return
	}

	fmt.Println("Starting SQL Alerts...")
	cmd = exec.Command(os.Args[0])
	err := cmd.Start()
	if err != nil {
		log.Fatalf("Failed to start SQL Alerts: %v", err)
	}
	fmt.Println("SQL Alerts started successfully.")
}

func stop() {
	cmd := exec.Command("pgrep", "-f", os.Args[0])

	output, _ := cmd.Output()
	if len(output) == 0 {
		fmt.Println("Nothing to stop")
		return
	}

	fmt.Println("Stopping sqlal...")
	out, err := exec.Command("pkill", "-f", os.Args[0]).CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to stop sqlal: %v", err)
	}
	fmt.Print(string(out))
	fmt.Println("sqlal stopped.")
}

func restart() {
	stop()
	start()
}

func config() {
	p := tea.NewProgram(internal.InitialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
	os.Exit(0)
}
