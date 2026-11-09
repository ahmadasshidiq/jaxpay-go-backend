package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"jaxpay-go-backend/libs/config"
	"jaxpay-go-backend/libs/cryptography"
)

func main() {
	user := flag.String("user", "admin", "user id (subject)")
	access := flag.String("access", "*", `hak akses dipisah koma, mis. "clients:*,payments:read"`)
	ttl := flag.Duration("ttl", 24*time.Hour, "masa berlaku token")
	flag.Parse()

	if config.Get().JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}
	t, e := cryptography.IssueAdminToken(*user, strings.Split(*access, ","), *ttl)
	if e != nil {
		log.Fatal(e)
	}
	fmt.Println(t)
}
