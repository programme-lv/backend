package testdb

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type flywayLogConsumer struct{}

func (consumer *flywayLogConsumer) Accept(l testcontainers.Log) {
	fmt.Print(string(l.Content))
}

// exectues flyway:21 on host network with migrationsDir mounted to /flyway/flyway-migrations
func execFlywayContainer(networkName string, networkAlias string, migrationDir string, dbHost string, dbPort string, dbName string, dbUser string, dbPassword string) error {
	args := []string{
		fmt.Sprintf("-url=jdbc:postgresql://%s:%s/%s", dbHost, dbPort, dbName),
		fmt.Sprintf("-user=%s", dbUser),
		fmt.Sprintf("-password=%s", dbPassword),
		"-connectRetries=5",
		"-locations=filesystem:/flyway/flyway-migrations",
		"migrate",
	}

	req := testcontainers.ContainerRequest{
		Image: "flyway/flyway:9.21",
		Mounts: testcontainers.ContainerMounts{
			testcontainers.ContainerMount{
				Source: testcontainers.GenericBindMountSource{
					HostPath: migrationDir,
				},
				Target: "/flyway/flyway-migrations",
			},
		},
		WaitingFor: wait.ForAll(wait.ForExit()),
		Cmd:        args,
		Networks:   []string{networkName},
		NetworkAliases: map[string][]string{
			networkName: {networkAlias},
		},
	}

	c, err := testcontainers.GenericContainer(context.Background(),
		testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          false,
		})
	if err != nil {
		return err
	}

	err = c.Start(context.Background())
	if err != nil {
		log.Printf("Error starting flyway container: %v", err)
	}
	cLogs, err := c.Logs(context.Background())
	if err != nil {
		log.Printf("Error getting flyway container logs: %v", err)
	}
	logs, err := io.ReadAll(cLogs)
	if err != nil {
		log.Printf("Error reading flyway container logs: %v", err)
	}
	log.Println(string(logs))
	return err
}
