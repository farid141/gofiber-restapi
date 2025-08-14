package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/farid141/go-rest-api/config"
	_ "github.com/go-sql-driver/mysql"
	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		app := fiber.New(fiber.Config{
			Prefork:       true,
			CaseSensitive: true,
			StrictRouting: true,
			ServerHeader:  "Fiber",
			AppName:       "Test App",
		})

		file, _ := os.OpenFile("application.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		log.SetOutput(file)

		// Login route
		app.Post("/login", login)

		// Unauthenticated route
		app.Get("/", func(c *fiber.Ctx) error {
			log.Info("hay")
			return c.SendString("Hello, World!")
		})

		// JWT Middleware
		app.Use(jwtware.New(jwtware.Config{
			SigningKey:  jwtware.SigningKey{Key: []byte("secret")},
			TokenLookup: "cookie:token", // 👈 look in cookie named "token"
		}))

		// Restricted Routes
		app.Get("/restricted", restricted)

		app.Listen(":3000")
	}()

	// Goroutine untuk baca input terminal
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("Enter command (up/down/exit): ")
			cmd, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error reading input:", err)
				return
			}
			cmd = strings.TrimSpace(cmd)

			if cmd == "" {
				continue // skip kalau kosong
			}

			dsn := fmt.Sprintf(
				"mysql://%s:%s@tcp(%s:%d)/%s",
				// "mysql://%s:%s@tcp(%s:%d)/%s",
				cfg.DBUser,
				cfg.DBPassword,
				cfg.DBHost,
				cfg.DBPort,
				cfg.DBName,
			)
			fmt.Println(dsn)
			switch cmd {
			case "up":
				runMigration(dsn, "file://db/migrations", true)
			case "down":
				runMigration(dsn, "file://db/migrations", false)
			case "exit":
				fmt.Println("Shutting down...")
				os.Exit(0)
			default:
				fmt.Println("Unknown command:", cmd)
			}
		}
	}()

	// Block supaya main nggak selesai
	select {}
}

func login(c *fiber.Ctx) error {
	fmt.Println("login")
	user := c.FormValue("user")
	pass := c.FormValue("pass")

	// Throws Unauthorized error
	if user != "john" || pass != "doe" {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	// Create the Claims
	claims := jwt.MapClaims{
		"name":  "John Doe",
		"admin": true,
		"exp":   time.Now().Add(time.Hour * 72).Unix(),
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token and send it as response.
	t, err := token.SignedString([]byte("secret"))
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	c.Cookie(&fiber.Cookie{Name: "token", Value: t})
	return c.JSON(fiber.Map{"token": t})
}

func restricted(c *fiber.Ctx) error {
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	name := claims["name"].(string)
	return c.SendString("Welcome " + name)
}

func runMigration(dbURL, migrationsPath string, up bool) {
	m, err := migrate.New(migrationsPath, dbURL)
	if err != nil {
		fmt.Println("Migration init error:", err)
		return
	}

	if up {
		err = m.Up()
	} else {
		err = m.Down()
	}

	if err != nil && err != migrate.ErrNoChange {
		fmt.Println("Migration error:", err)
		return
	}

	fmt.Println("Migration successful")
}
