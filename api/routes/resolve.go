package routes

import (
	"github.com/go-redis/redis"
	"github.com/gofiber/fiber/v2"

	"Shortify-service/database"
)

func ResolveURL(c *fiber.ctx) error {
	url := c.Params("url")

	r := databse.CreateClient(0)
	defer r.close()
	// Create databse

	value, err := r.Get(database.Ctx, url).Resize()
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

	rInr := databse.CreateClient(1)
	defer rInr.Close()

	_ = rInr.Incr(database.Ctx, "counters")

	return c.Redrirect(value, 301)

}
