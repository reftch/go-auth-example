package handler

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
)

var e *echo.Echo

func init() {
	godotenv.Load()

	// init session
	key := os.Getenv("SESSION_SECRET") // Replace with your SESSION_SECRET or similar
	maxAge := 86400 * 30               // 30 days
	isProd := false                    // Set to true when serving over https

	store := sessions.NewCookieStore([]byte(key))
	store.MaxAge(maxAge)
	store.Options.Path = "/"
	store.Options.HttpOnly = true // HttpOnly should always be enabled
	store.Options.Secure = isProd

	gothic.Store = store

	// init goth providers
	goth.UseProviders(
		github.New(
			os.Getenv("GITHUB_KEY"),
			os.Getenv("GITHUB_SECRET"),
			os.Getenv("GITHUB_CALLBACK"),
		),
	)

	e = echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Static("public"))

	e.GET("/", func(c echo.Context) error {
		return c.HTML(http.StatusOK, indexTemplate)
	})

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	e.GET("/auth/:provider", func(c echo.Context) error {
		provider := c.Param("provider")
		if provider == "" {
			return c.String(http.StatusBadRequest, "Provider not specified")
		}

		q := c.Request().URL.Query()
		q.Add("provider", c.Param("provider"))
		c.Request().URL.RawQuery = q.Encode()

		req := c.Request()
		res := c.Response().Writer
		if gothUser, err := gothic.CompleteUserAuth(res, req); err == nil {
			return c.JSON(http.StatusOK, gothUser)
		}
		gothic.BeginAuthHandler(res, req)
		return nil
	})

	e.GET("/auth/:provider/callback", func(c echo.Context) error {
		req := c.Request()
		res := c.Response().Writer
		user, err := gothic.CompleteUserAuth(res, req)
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, user)
	})

}

func Handler(w http.ResponseWriter, r *http.Request) {
	e.ServeHTTP(w, r)
}

func Local() {
	log.Println("listening on http://localhost:8080")
	e.Logger.Fatal(e.Start(":8080"))
}

var indexTemplate = `
<div>
    <p><a href="/auth/github">Log in with Github</a></p>
</div>`
