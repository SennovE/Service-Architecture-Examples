package provider

import (
	"context"
	"encoding/json"
	"flights/internal/service"
	"iter"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type Cache struct {
	rdb *redis.Client
}

var setFlightScript = redis.NewScript(`
local dataKey = KEYS[1]
local versionKey = KEYS[2]

local newData = ARGV[1]
local newVersion = tonumber(ARGV[2])
local ttlSeconds = tonumber(ARGV[3])

local currentVersion = tonumber(redis.call('GET', versionKey) or '-1')

if currentVersion >= newVersion then
	return 0
end

redis.call('SET', dataKey, newData, 'EX', ttlSeconds)
redis.call('SET', versionKey, newVersion, 'EX', ttlSeconds)

return 1
`)

var removeFlightScript = redis.NewScript(`
local dataKey = KEYS[1]
local versionKey = KEYS[2]

local newVersion = tonumber(ARGV[1])
local ttlSeconds = tonumber(ARGV[2])

local currentVersion = tonumber(redis.call('GET', versionKey) or '-1')

if currentVersion >= newVersion then
	return 0
end

redis.call('DEL', dataKey)
redis.call('SET', versionKey, newVersion - 1, 'EX', ttlSeconds)

return 1
`)

func flightKey(flightID uuid.UUID) string {
	return "flight:" + flightID.String()
}

func flightSearchKey(origin, destination string, date *time.Time) string {
	key := "search:" + origin + ":" + destination
	if date != nil {
		key += ":" + date.UTC().Format("2006-01-02")
	}
	return key
}

func flightVersionKey(flightID uuid.UUID) string {
	return "flight:" + flightID.String() + ":version"
}

func (c *Cache) GetFlight(ctx context.Context, flightID uuid.UUID) *service.Flight {
	res, err := c.rdb.Get(ctx, flightKey(flightID)).Result()
	switch err {
	case nil:
		var flight service.Flight
		err = json.Unmarshal([]byte(res), &flight)
		if err != nil {
			return nil
		}
		log.Printf("Cache HIT for flight with ID = %s\n", flightID.String())
		return &flight
	case redis.Nil:
		log.Printf("Cache MISS for flight with ID = %s\n", flightID.String())
		return nil
	default:
		return nil
	}
}

func (c *Cache) SetFlight(flight *service.Flight, ttl time.Duration) {
	flightJSON, err := json.Marshal(flight)
	if err != nil {
		return
	}
	setFlightScript.Run(
		context.Background(),
		c.rdb,
		[]string{
			flightKey(flight.ID),
			flightVersionKey(flight.ID),
		},
		flightJSON,
		flight.Version,
		int(ttl.Seconds()),
	)
	log.Println("Updated cache for flight")
}

func (c *Cache) RemoveFlight(flightID uuid.UUID, version uint64, ttl time.Duration) {
	removeFlightScript.Run(
		context.Background(),
		c.rdb,
		[]string{
			flightKey(flightID),
			flightVersionKey(flightID),
		},
		version,
		int(ttl.Seconds()),
	)
}

func (c *Cache) GetSearchFlights(
	ctx context.Context, origin, destination string, date *time.Time) iter.Seq[*service.Flight] {
	res, err := c.rdb.Get(ctx, flightSearchKey(origin, destination, date)).Result()
	if err != nil {
		log.Println("Cache MISS for flights")
		return nil
	}
	var ids []uuid.UUID
	err = json.Unmarshal([]byte(res), &ids)
	if err != nil {
		return nil
	}
	flights := make([]*service.Flight, len(ids))
	for i, id := range ids {
		flight := c.GetFlight(ctx, id)
		if flight == nil {
			return nil
		}
		flights[i] = flight
	}
	log.Println("Cache FULL HIT for flights")
	return func(yield func(*service.Flight) bool) {
		for _, flight := range flights {
			if !yield(flight) {
				return
			}
		}
	}
}

func (c *Cache) SetSearchFlights(origin, destination string, date *time.Time, flightsChan <-chan *service.Flight, ttl time.Duration) {
	var ids []uuid.UUID
	for flight := range flightsChan {
		ids = append(ids, flight.ID)
		go c.SetFlight(flight, ttl)
	}
	idsJSON, err := json.Marshal(ids)
	if err != nil {
		return
	}
	c.rdb.Set(context.Background(), flightSearchKey(origin, destination, date), idsJSON, ttl)
	log.Println("Updated cache for flights")
}
