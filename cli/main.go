package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"github.com/gocql/gocql"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	var startDateStr, endDateStr, logLevel, outputFile, messageSubstr string
	var attributes []string

	flag.StringVar(&startDateStr, "start", "", "Start date in format YYYY-MM-DD (optional)")
	flag.StringVar(&endDateStr, "end", "", "End date in format YYYY-MM-DD (optional)")
	flag.StringVar(&logLevel, "loglevel", "", "Log level to filter (optional)")
	flag.StringVar(&messageSubstr, "message", "", "Substring in message to filter (optional)")
	flag.StringVar(&outputFile, "output",
		fmt.Sprintf("log_%s.csv", strings.Replace(time.Now().Format(time.DateTime), " ", "_", 1)),
		"Output CSV file",
	)
	flag.Var((*stringArrayFlag)(&attributes), "attr", "Attributes to filter (can be used multiple times)")

	help := flag.Bool("help", false, "Display help information")

	flag.Parse()

	if *help {
		flag.Usage()
		return
	}

	startDate, _ := parseDate(startDateStr)
	endDate, _ := parseDate(endDateStr)

	logs := queryLogs(startDate, endDate, logLevel, attributes, messageSubstr)

	if err := writeCSV(outputFile, logs); err != nil {
		log.Fatal(err)
	}
}

// Custom flag for multiple attributes
type stringArrayFlag []string

func (i *stringArrayFlag) String() string {
	return fmt.Sprint(*i)
}

func (i *stringArrayFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// createCluster configures and returns a ScyllaDB cluster connection.
func createCluster(consistency gocql.Consistency, keyspace string, hosts ...string) *gocql.ClusterConfig {
	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = keyspace
	cluster.Consistency = consistency
	cluster.Timeout = 5 * time.Second
	cluster.RetryPolicy = &gocql.ExponentialBackoffRetryPolicy{
		Min:        time.Second,
		Max:        10 * time.Second,
		NumRetries: 5,
	}
	cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())
	return cluster
}

// LogEntry represents log data
type LogEntry struct {
	Timestamp  time.Time
	Attributes string
	LogLevel   string
	Message    string
}

// queryLogs retrieves log entries from ScyllaDB based on the specified filters.
func queryLogs(startDate, endDate int64, logLevel string, attributes []string, messageSubstr string) []LogEntry {
	cluster := createCluster(gocql.Quorum, "log_storage", "127.0.0.1")
	session, err := gocql.NewSession(*cluster)
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}
	defer session.Close()

	query := buildQuery(startDate, endDate, logLevel, attributes, messageSubstr)

	var logs []LogEntry
	iter := session.Query(query).Iter()
	m := make(map[string]interface{})

	for iter.MapScan(m) {
		logEntry := LogEntry{
			Timestamp:  m["timestamp"].(time.Time),
			Attributes: fmt.Sprintf("%v", m["attributes"]),
			LogLevel:   m["log_level"].(string),
			Message:    m["message"].(string),
		}
		m = map[string]interface{}{}
		logs = append(logs, logEntry)
	}

	return logs
}

// buildQuery constructs a CQL query string based on the specified filters.
func buildQuery(startDate, endDate int64, logLevel string, attributes []string, messageSubstr string) string {
	query := "SELECT timestamp, attributes, log_level, message FROM logs"
	var conditions []string

	if startDate != -1 {
		conditions = append(conditions, fmt.Sprintf("timestamp >= %d", startDate))
	}
	if endDate != -1 {
		conditions = append(conditions, fmt.Sprintf("timestamp <= %d", endDate))
	}
	if logLevel != "" {
		conditions = append(conditions, fmt.Sprintf("log_level = '%s'", logLevel))
	}
	if messageSubstr != "" {
		conditions = append(conditions, fmt.Sprintf("message LIKE '%%%s%%'", messageSubstr))
	}
	for _, attr := range attributes {
		conditions = append(conditions, fmt.Sprintf("attributes CONTAINS KEY '%s'", attr))
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for _, condition := range conditions[1:] {
			query += " AND " + condition
		}
	}

	query += " ALLOW FILTERING"
	return query
}

// parseDate converts a date string to a Unix timestamp (int64). Returns -1 if the date string is empty.
func parseDate(dateStr string) (int64, error) {
	if dateStr == "" {
		return -1, nil
	}
	parsedDate, err := time.Parse("2006-01-02 15:04", dateStr)
	if err != nil {
		parsedDate, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			log.Fatalf("Invalid date format: %v. Use 'YYYY-MM-DD' or 'YYYY-MM-DD hh:mm'.", err)
		}
	}
	return parsedDate.Unix(), nil
}

// writeCSV writes the logs to a CSV file.
func writeCSV(fileName string, logs []LogEntry) error {
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err = writer.Write([]string{"Timestamp", "Attributes", "Log Level", "Message"}); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write log entries
	for _, entry := range logs {
		if err = writer.Write([]string{
			entry.Timestamp.Format(time.RFC3339),
			entry.Attributes,
			entry.LogLevel,
			entry.Message,
		}); err != nil {
			return fmt.Errorf("failed to write log entry: %w", err)
		}
	}

	return nil
}
