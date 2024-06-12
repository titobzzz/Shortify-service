package routes

import (
	"os"
	"strconv"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"Shortify-service/database"
	"Shortify-service/helpers"
)

type request struct {
	URL         string        `json:"url"`
	CustomShort string        `json:"short"`
	Expiry      time.Duration `json:"expiry"`
}

type response struct {
	URL             string        `json:"url"`
	CustomShort     string        `json:"short"`
	Expiry          time.Duration `json:"expiry"`
	XRateRemaining  int           `json:"rate_limit"`
	XRateLimitReset time.Duration `json:"rate_limit_reset"`
}

func ShortenURL(c *fiber.Ctx) error {

	body := new(request)
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cannot parse JSON",
		})
	}

	//create a rate limiting filter
	r2 := database.CreateClient(1)
	defer r2.Close()
	val, err := r2.Get(database.Ctx, c.IP()).Result()
	if err == redis.Nil {
		_ = r2.Set(database.Ctx, c.IP(), os.Getenv("API_QUOTA"), 30*60*time.Second).Err()
	} else {
		val, _ = r2.Get(database.Ctx, c.IP()).Result()
		valInt, _ := strconv.Atoi(val)
		if valInt <= 0 {
			limit, _ := r2.TTL(database.Ctx, c.IP()).Result()
			if limit == -1 {
				// case where the key has no expiration time
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
					"error":            "Rate limit exceeded",
					"rate_limit_reset": -1,
				})
			}

			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":            "Rate limit exceeded",
				"rate_limit_reset": limit / time.Nanosecond / time.Minute,
			})
		}
	}
	//  else {
	// 	val, _ = r2.Get(database.Ctx, c.IP()).Result()
	// 	valInt, _ := strconv.Atoi(val)
	// 	if valInt <= 0 {
	// 		limit, _ := r2.TTL(database.Ctx, c.IP()).Result()
	// 		if limit == -1 {
	// 			// case where the key has no expiration time
	// 			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
	// 				"error":            "Rate limit exceeded",
	// 				"rate_limit_reset": -1,
	// 			})
	// 		}

	// 		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
	// 			"error":            "Rate limit exceeded",
	// 			"rate_limit_reset": limit / time.Nanosecond / time.Minute,
	// 		})
	// 	}
	// }

	//validate input as actual URL
	if !govalidator.IsURL(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "canot validate",
		})
	}

	//handle localhost issues
	if !helpers.RemoveDomainError(body.URL) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "put valide url",
		})
	}

	//enforce http and ssl

	body.URL = helpers.EnforceHTTP(body.URL)

	var id string
	if body.CustomShort == "" {
		id = uuid.New().String()[:6]
	} else {
		id = body.CustomShort
	}

	r := database.CreateClient(0)
	defer r.Close()

	val, _ = r.Get(database.Ctx, id).Result()
	if val != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "URL Custom short already used",
		})
	}
	if body.Expiry == 0 {
		body.Expiry = 24 // default expiry of 24 hours
	}
	err = r.Set(database.Ctx, id, body.URL, body.Expiry*time.Second).Err()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "unable to connect to server",
		})
	}
	//for response
	resp := response{
		URL:             body.URL,
		CustomShort:     "",
		Expiry:          body.Expiry,
		XRateRemaining:  10,
		XRateLimitReset: 3,
	}
	r2.Decr(database.Ctx, c.IP())
	val, _ = r2.Get(database.Ctx, c.IP()).Result()

	resp.XRateRemaining, _ = strconv.Atoi(val)
	ttl, _ := r2.TTL(database.Ctx, c.IP()).Result()
	resp.XRateLimitReset = ttl / time.Nanosecond / time.Minute

	resp.CustomShort = os.Getenv("DOMAIN") + "/" + id
	return c.Status(fiber.StatusOK).JSON(resp)

}
