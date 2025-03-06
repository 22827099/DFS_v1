package logger

import "fmt"

func main() {
	config := Config{
		Level:     "INFO",
		Format:    "json",
		Output:    "both",
		FilePath:  "logs/app.log",
		MaxSizeMB: 10,
		KeepDays:  7,
	}

	log := NewLogger(config)

	log.log("INFO", "Application started", map[string]interface{}{
		"module": "main",
	})

	fmt.Println("Log written")
}
