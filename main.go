package main

import (
	"log"

	internalApp "hvac-system/internal/app"
	"hvac-system/pkg/app"

	_ "hvac-system/migrations"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
)

func main() {
	pb := pocketbase.New()

	// 1. Migrations
	migratecmd.MustRegister(pb, pb.RootCmd, migratecmd.Config{
		Automigrate: true,
	})

	// 2. Initialize DI Container (Single source of truth for all dependencies)
	container, err := internalApp.NewContainer(pb)
	if err != nil {
		log.Fatal("Error initializing container:", err)
	}

	// 3. Register Routes (passes only the Container)
	app.RegisterRoutes(pb, container)

	if err := pb.Start(); err != nil {
		log.Fatal(err)
	}
}
