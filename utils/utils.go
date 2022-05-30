package utils

import (
	"log"
	"net/http"
	"os"
	"github.com/joho/godotenv"
)

// use godot package to load/read the .env file and
// return the value of the key
func goDotEnvVariable(key string) string {

	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	return os.Getenv(key)
}

func GetApiUrl(query string) string {
	req, err := http.NewRequest("GET", "https://api.openweathermap.org/data/2.5/weather", nil)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	API_KEY := goDotEnvVariable("API_KEY")

	q := req.URL.Query()
	q.Add("q", query)
	q.Add("units", "metric")
	q.Add("APPID", API_KEY)
	req.URL.RawQuery = q.Encode()

	// fmt.Println(req.URL.String())

	url := req.URL.String()
	return url
}