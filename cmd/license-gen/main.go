package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// MUST match the one in pkg/middleware/settings.go
const LicenseSecretKey = "HVAC_SECURE_V1_@992834_DIGITAL_SEAL_X"

func main() {
	var customerName string
	var days int

	// Command line flags (optional, can also use interactive mode)
	flag.StringVar(&customerName, "name", "", "Customer Name")
	flag.IntVar(&days, "days", 0, "Validity days")
	flag.Parse()

	reader := bufio.NewReader(os.Stdin)

	if customerName == "" {
		fmt.Print("Enter Customer Name: ")
		name, _ := reader.ReadString('\n')
		customerName = strings.TrimSpace(name)
	}

	if days == 0 {
		fmt.Print("Enter Validity Days (e.g. 365): ")
		fmt.Scanln(&days)
	}

	if customerName == "" || days <= 0 {
		fmt.Println("Error: Invalid input")
		os.Exit(1)
	}

	// Calculate Expiry
	expiryDate := time.Now().AddDate(0, 0, days)

	// Create Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": customerName,
		"exp": expiryDate.Unix(),
		"iat": time.Now().Unix(),
		"iss": "HVAC-System-Gen-Tool",
	})

	// Sign
	tokenString, err := token.SignedString([]byte(LicenseSecretKey))
	if err != nil {
		fmt.Printf("Error generating token: %v\n", err)
		return
	}

	fmt.Println("\n===========================================")
	fmt.Println("       HVAC SYSTEM - DIGITAL SEAL")
	fmt.Println("===========================================")
	fmt.Printf("Customer: %s\n", customerName)
	fmt.Printf("Expires : %s (%d days)\n", expiryDate.Format("2006-01-02"), days)
	fmt.Println("-------------------------------------------")
	fmt.Println("LICENSE KEY (Copy below line):")
	fmt.Println(tokenString)
	fmt.Println("===========================================")
}
