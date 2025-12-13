package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/sherlock/service/internal/database"
	"github.com/sherlock/service/internal/types"
)

func main() {
	var (
		email    = flag.String("email", "", "Super admin email (required)")
		password = flag.String("password", "", "Super admin password (required)")
		name     = flag.String("name", "", "Super admin name (required)")
		dbURL    = flag.String("db", "", "Database URL (required)")
	)
	flag.Parse()

	if *email == "" || *password == "" || *name == "" || *dbURL == "" {
		fmt.Println("Usage: create-admin -email <email> -password <password> -name <name> -db <database_url>")
		fmt.Println("\nExample:")
		fmt.Println("  create-admin -email admin@example.com -password secure123 -name 'Super Admin' -db 'postgres://user:pass@localhost/sherlock?sslmode=disable'")
		os.Exit(1)
	}

	// Connect to database
	db, err := database.New(*dbURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Check if user already exists
	existingUser, err := db.GetUserByEmail(*email)
	if err == nil && existingUser != nil {
		fmt.Printf("User with email %s already exists\n", *email)
		if existingUser.Role == types.RoleSuperAdmin {
			fmt.Println("✅ User is already a super admin")
			os.Exit(0)
		} else {
			fmt.Println("User exists but is not a super admin. Updating role...")
			err := db.UpdateUserRole(existingUser.ID, types.RoleSuperAdmin)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to update user role")
			}
			fmt.Printf("✅ User role updated to super admin!\n\n")
			fmt.Printf("Email: %s\n", existingUser.Email)
			fmt.Printf("Name: %s\n", existingUser.Name)
			fmt.Printf("Role: super_admin\n")
			fmt.Printf("ID: %s\n\n", existingUser.ID)
			os.Exit(0)
		}
	}

	// Create super admin user (no org_id for super admin)
	user, err := db.CreateUser(*email, *password, *name, types.RoleSuperAdmin, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create super admin user")
	}

	fmt.Printf("✅ Super admin created successfully!\n\n")
	fmt.Printf("Email: %s\n", user.Email)
	fmt.Printf("Name: %s\n", user.Name)
	fmt.Printf("Role: %s\n", user.Role)
	fmt.Printf("ID: %s\n\n", user.ID)
	fmt.Println("You can now log in with these credentials.")
}
