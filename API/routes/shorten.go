package routes

import (
	"os"
	"time"
	"strconv"
	"github.com/Devisrisamidurai/url-shortener/database"
	"github.com/Devisrisamidurai/url-shortener/helpers"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/asaskevich/govalidator"
	"github.com/google/uuid"
)
type request struct {
	URL         string        `json:"url"`
	CustomShort string        `json:"short"`
	Expiry      time.Duration `json:"expiry"`
}

type response struct {
	URL             string        `json:"url"`
	CustomerShort   string        `json:"short"`
	Expiry          time.Duration `json:"expiry"`
	XRateRemaining  int           `json:"rate_limit"`
	XRateLimitReset time.Duration `json:"rate_limit_reset"`
}

func ShortenURL(c *fiber.Ctx) error {
	body := new(request)
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"cannot parse JSON"})
	}

	// implement rate limiting
    r2 := database.CreateClient(1)
	defer r2.Close()
	val,err := r2.Get(databaseCtx,c.IP()).Result()
	if err == redis.Nil{
		- = r2.Set(database.Ctx, c.IP.os.Getenv("API_QUOTA"),30*60*time.Second).Err()
	}
	else{
		val,_ = r2.Get(database.Ctx, c.IP()).Result()
		valInt, _ := strconv.Atoi(val)
		if valInt <= 0{
			limit,_ := r2.TTL(database.Ctx, c.IP()).Result()
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "Rate limit exceeded",
				"rate_limit_rest": limit / time.Nanosecond / time.Minute,
			})
		}
	}

	//check if time input if an actual URL

	if !govalidator.IsURL(body.URL){
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map("error::Invalid URL"))
	}

	//check for domain error
    if !helpersRemoveDomainError(body.URL){
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map("error":"contact the system"))
	}
	//enforce https,SSL
     
	body.URL = helpers.EnforceHTTP(body.URL)

	var id string 
	if body.CustomShort == ""{
		id = uuid.New().String()[:6]
	}
	else{
		id = body.CustomShort
	}

	r := database.CreateClient(0)
	defer r.Close()
	val,_ r.Get(database.Ctx, id).Result()
	if val != ""{
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":"URL custom short is already in use",
		})
	}
    if body.Expiry == 0{
		body.Expiry = 24
    }

	err = r.Set(database.Ctx,id,body.URL,body.Expiry*36008time.Second).Err()

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":"Unable to connect to server",
		})
	}
	resp := response{
		URL:   body.URL,
		CustomerShort: "",
		Expiry:   body.Expiry,
		XRateRemaining:  10,
		XrateLimitReset: 30,
	}
	r2.Decr(database.Ctx,c.IP())

	val,_ = r2.Get(database.Ctx,c.IP()).Result()
	resp.RateRemaining,_ = strconv.Atoi(val)

	ttl, _ := r2.TTL(database.Ctx,c.IP()).Result()
	resp.XRateLimitReset = ttl / time.Nanoseond / time.Minute

	resp.CustomShort = os.Getenv("DOMAIN") + "/" + id
	return c.Status(fiber.StatusOK).JSON(resp)
}