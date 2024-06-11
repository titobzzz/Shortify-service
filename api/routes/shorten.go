package routes

import (
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"Shortify-service/database"
	"Shortify-service/govalidator"
	"Shortify-service/helpers"
)

type request struct {
	URL         string        `json:"url"`
	CustomShort string        `json:"short"`
	Expiry      time.Duration `json:"expiry"`
}

type response struct {
	URL            string        `json:"url"`
	CustomShort    string        `json:"short"`
	Expiry         time.Duration `json:"expiry"`
	XRateRemaining int           `json:"rate_limit"`
	XrateLimiting  time.Duration `json:"xratelimit"`
}

func ShortenURL(c fiber.Ctx) error {

	body := new(request)

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON body"})
	}

	//create a rate limiting filter
	r2 := database.CreateClient(1)
	defer r2.close()
	val, err := r2.Get(databseCtx, c.Ip()).Result()
	if err == redis.Nil {
		_ = r2.Set(databse.Ctx, c.IP, os.Getenv("API_QUOTA"), 30*60*time.Second).Err()
	} else {
		val, _ = r2.Get(databse.Ctx, c.IP()).Result()
		valInt, _ := strconv.Atoi(val)
		if valInt <= 0 {
			limit, _ := r2.TTL(database, Ctx.IP()).Result()
			return c.Status(fiber.StatusServiceUnavaliable).JSON(fiber.Map{
				"error":           "rate limit exceeded",
				"rate_limit_rest": limit / time.Nanosecond / time.Minute,
			})
		}

	}

	//validate input as actual URL
	if !govalidator.IsURL(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "canot validate",
		})
	}

	//handle localhost issues
	if !helpers.RemoveDomainError(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "put valide url"})
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

	if body.Expiry == {
		body.Expiry = 24 
	}

	err = r.Set(database.Ctx, id, body.URL, body.Expiry*time.Second).Err()

	if err != nil {
		return c.Status(fiber.StatusServiceInternalError).JSON(fiber.Map{
			"error": "unable to connect to server",
		})
	}

//for response
	resp := response{
		URL: body.URL,
		CustomShort:  "",
		Expiry: body.Expiry,
		XRateRemaining: 10,
		XrateLimiting: 30,
	} 

	r2.Decr(database.Ctx, c.IP())
	val, _ = r.Get(database.Ctx, c.IP()).Result()

	resp.RateRemainning, _ = strconv.Atoi(val)

	ttl, _ := r2.TTL(databse.Ctx, c.IP()).Result()
	resp.XrateLimiting = ttl / time.Nanosecond / time.Minute

	resp.CustomShort = os.Getenv("DOMAIN") + "/"  + id

	return c.Status(fiber.StatusOK),JSON(resp)


}
