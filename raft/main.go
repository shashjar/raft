package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
)

const (
	MINIMUM_PORT = 0
	MAXIMUM_PORT = 65535
)

type ServerAddress struct {
	id   int
	host string
	port int
}

func configureLogger() {
	log.SetFlags(0)
}

func parseClusterMembers(clusterStr string) (map[int]ServerAddress, error) {
	// Expected format for a 3-server cluster: "1:localhost:8081,2:localhost:8082,3:localhost:8083"
	members := make(map[int]ServerAddress)

	if clusterStr == "" {
		return nil, fmt.Errorf("cluster configuration is empty")
	}

	pairs := strings.Split(clusterStr, ",")
	for _, pair := range pairs {
		parts := strings.Split(pair, ":")
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid cluster member format: %s", pair)
		}

		serverID, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid server ID: %s", parts[0])
		}

		port, err := strconv.Atoi(parts[2])
		if err != nil || port < MINIMUM_PORT || port > MAXIMUM_PORT {
			return nil, fmt.Errorf("invalid port number: %s", parts[2])
		}

		members[serverID] = ServerAddress{
			id:   serverID,
			host: parts[1],
			port: port,
		}
	}

	return members, nil
}

func parseCommandLineArguments() (int, int, map[int]ServerAddress) {
	serverID := flag.Int("id", -1, "Server ID (must be unique in cluster)")
	clusterStr := flag.String("cluster", "", "Cluster members in format '1:localhost:8081,2:localhost:8082,3:localhost:8083'")

	flag.Parse()

	if *serverID < 1 {
		log.Fatalf("Server ID must be provided and greater than 0")
	}

	clusterMembers, err := parseClusterMembers(*clusterStr)
	if err != nil {
		log.Fatalf("Failed to parse cluster configuration: %v", err)
	}

	if _, exists := clusterMembers[*serverID]; !exists {
		log.Fatalf("Server ID %d not found in cluster configuration", *serverID)
	}

	return *serverID, clusterMembers[*serverID].port, clusterMembers
}

func main() {
	configureLogger()
	serverID, port, clusterMembers := parseCommandLineArguments()

	log.Printf("Starting Raft server ID=%d on port=%d", serverID, port)
	log.Printf("Cluster size: %d", len(clusterMembers))

	StartServer(serverID, clusterMembers)
}
