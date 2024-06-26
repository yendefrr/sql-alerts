package internal

import (
	"encoding/json"
	"os"
)

type Config struct {
	Database             DatabaseConfig `json:"database"`
	Queries              []QueryConfig  `json:"queries"`
	BaseNotificationURL  string         `json:"baseNotificationUrl"`
	NotificationMessage  string         `json:"notificationMessage"`
	CheckIntervalSeconds int            `json:"checkIntervalSeconds"`
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
	NotificationURL string `json:"notificationUrl"`
	Disabled        bool   `json:"disabled"`
}

func NewDefaultConfig() Config {
	return Config{
		Database: DatabaseConfig{
			Username: "",
			Password: "",
			Host:     "",
			Port:     "",
			Name:     "",
		},
		Queries: []QueryConfig{
			{
				Name:            "Query 1",
				Query:           "SELECT id FROM table",
				NotificationURL: "https://ntfy.sh/sqlal",
				Disabled:        false,
			},
		},
		BaseNotificationURL:  "https://ntfy.sh/sqlal",
		NotificationMessage:  "New %d rows",
		CheckIntervalSeconds: 60,
	}
}

func (c Config) SaveToFile(filename string) error {
	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) LoadFromFile(filename string) error {
	fileData, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	err = json.Unmarshal(fileData, c)
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) GetQueryNames() []string {
	var names []string

	for _, query := range c.Queries {
		names = append(names, query.Name)
	}
	return names
}

func (c *Config) AddQuery(newQuery QueryConfig) {
	c.Queries = append(c.Queries, newQuery)
}

func (c *Config) UpdateQuery(index int, newQuery QueryConfig) {
	if index < 0 || index >= len(c.Queries) {
		return // index out of range
	}
	c.Queries[index] = newQuery
}

func (c *Config) DeleteQueryByIndex(index int) {
	if index < 0 || index >= len(c.Queries) {
		return // index out of range
	}
	c.Queries = append(c.Queries[:index], c.Queries[index+1:]...)
}

func (c *Config) UpdateDB(newDB DatabaseConfig) {
	c.Database = newDB
}

func (c *Config) UpdateSettings(newSettings *Config) {
	*c = *newSettings
}
