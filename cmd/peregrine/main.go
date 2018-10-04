package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/tba"

	"github.com/Pigmice2733/peregrine-backend/internal/config"
	"github.com/Pigmice2733/peregrine-backend/internal/server"
)

func main() {
	var basePath = flag.String("basePath", ".", "Path to the etc directory where the config file is.")

	flag.Parse()

	c, err := config.Open(*basePath)
	if err != nil {
		fmt.Printf("Error: opening config: %v\n", err)
		return
	}

	tba := tba.Service{
		URL:    c.TBA.URL,
		APIKey: c.TBA.APIKey,
	}

	s, err := store.New(c.Database)
	if err != nil {
		fmt.Printf("Error: unable to connect to postgres server: %v\n", err)
		return
	}

	if c.SeedUser != nil {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(c.SeedUser.Password), bcrypt.DefaultCost)
		if err != nil {
			fmt.Printf("Error: creating hashed password")
			return
		}

		u := store.User{
			Username:       c.SeedUser.Username,
			HashedPassword: string(hashedPassword),
			FirstName:      c.SeedUser.FirstName,
			LastName:       c.SeedUser.LastName,
			Roles:          c.SeedUser.Roles,
		}

		err = s.CreateUser(u)
		if err == store.ErrExists {
			fmt.Printf("Error: seed user already exists\n")
		} else if err != nil {
			fmt.Printf("Error: unable to create seed user: %v\n", err)
			return
		}
	}

	year := c.Server.Year
	if year == 0 {
		year = time.Now().Year()
	}

	jwtSecret := make([]byte, 64)
	if _, err := rand.Read(jwtSecret); err != nil {
		fmt.Printf("Error: generating jwt secret: %v\n", err)
		return
	}

	server := server.New(
		tba,
		s,
		c.Server.HTTPAddress,
		c.Server.HTTPSAddress,
		c.Server.CertFile,
		c.Server.KeyFile,
		c.Server.Origin,
		jwtSecret,
		year,
	)

	if err := server.Run(); err != nil {
		fmt.Printf("Error: server.Run: %v\n", err)
	}
}
