package SignalingServer

import "os"

const DefaultPort = "4444"

// get port from environment variable if exists
func GetPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}
	return ":" + port
}
