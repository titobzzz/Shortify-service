package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	"Shortify-service/database"
)

func ResolveURL(c *fiber.Ctx) error {
	url := c.Params("url")

	r := database.CreateClient(0)
	defer r.Close()
	// Create databse

	value, err := r.Get(database.Ctx, url).Result()
	// try to get value from DB
	//ahndle error for not found and unable to connect
	if err == redis.Nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "short not found in the databse",
		})
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "cannot connect to Database",
		})
	}

	rInr := database.CreateClient(1)
	defer rInr.Close()

	_ = rInr.Incr(database.Ctx, "counters")

	return c.Redirect(value, 301)

}
