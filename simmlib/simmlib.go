package simmlib

import (
	"fmt"
	"os"
	"time"

	redisLib "github.com/garyburd/redigo/redis"
)

type (
	TimeResolution struct {
		resolution string
		timestamp  uint64
	}

	handlerVars struct {
		redisConnection *redisLib.Conn
	}
)

// !private
var (
	err    error
	hnVars handlerVars

	// time slices for graph
	resolutions = map[string]uint64{
		"min":   60,
		"5min":  300,
		"15min": 900,
		"hour":  3600,
		"day":   86400,
		"week":  86400 * 7,
		"month": 86400 * 30,
		"year":  86400 * 365,
	}

	RedisArgs = struct {
		RedisHost     string
		RedisPort     string
		RedisDb       string
		RedisPassword string
	}{
		RedisHost: DEFAULT_REDIS_HOST,
		RedisPort: DEFAULT_REDIS_PORT,
	}
)

const (
	DEFAULT_INCREMENT      = uint64(1)
	DEFAULT_RESOLUTION     = "5min"
	DEFAULT_REDIS_HOST     = "localhost"
	DEFAULT_REDIS_PORT     = "6379"
	DEFAULT_REDIS_DB       = ""
	DEFAULT_REDIS_PASSWORD = ""
)

func Check(err error) {
	if err != nil {
		panic(err)
		os.Exit(1)
	}
}

func Push(event string, increment uint64, now uint64) (interface{}, error) {

	redisConnection := *hnVars.redisConnection
	var key string
	redisConnection.Send("MULTI")
	for timeRes := range getTimeStampsForPush(now) {
		key = getEventKey(event, timeRes.resolution)
		err := redisConnection.Send("HINCRBY", key, timeRes.timestamp, increment)
		Check(err)
	}
	return redisConnection.Do("EXEC")
}

// RoundTime rounds given time according to given resolution which defined in resolutions hash map
func RoundTime(time uint64, resolution uint64) uint64 {
	return uint64(time - (time % resolution))
}

// getEventKey returns the event key according to given event name and resolution key
// !private
func getEventKey(event string, resolutionKey string) string {
	return fmt.Sprintf("simmetrica:%s:%s", event, resolutionKey)
}

// getTimeStampsForPush yields timestamps for each resolution which defined in resolutions hash map
// !private
func getTimeStampsForPush(now uint64) chan TimeResolution {
	if now == 0 {
		now = GetCurrentTimeStamp()
	}
	timeResolution := make(chan TimeResolution)
	go func() {
		for resolution, timestamp := range resolutions {
			timeResolution <- TimeResolution{resolution: resolution, timestamp: RoundTime(now, timestamp)}
		}
		close(timeResolution)
	}()

	return timeResolution
}

// GetCurrentTimeStamp returns current timestamp as type uint64
// !private
func GetCurrentTimeStamp() uint64 {
	return uint64(time.Now().Unix())
}
func Initialize() *redisLib.Conn {
	redisConnection, err := redisLib.Dial("tcp", fmt.Sprintf("%s:%s", RedisArgs.RedisHost, RedisArgs.RedisPort))
	hnVars.redisConnection = &redisConnection
	Check(err)
	return hnVars.redisConnection
}

func Uninitialize() func(redisConnection redisLib.Conn) {
	return func(redisConnection redisLib.Conn) {
		redisConnection.Close()
	}
}
