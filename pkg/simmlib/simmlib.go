package simmlib

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
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

	TstampValTuple struct {
		x uint64
		y uint8
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
	DEFAULT_REDIS_HOST     = "redis"
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

func Initialize() {
	redisConnection, err := redisLib.Dial("tcp", fmt.Sprintf("%s:%s", RedisArgs.RedisHost, RedisArgs.RedisPort))
	hnVars.redisConnection = &redisConnection
	Check(err)
}

func Uninitialize() {
	redisConnection := *hnVars.redisConnection
	redisConnection.Close()
}

// Push sends HINCRBY command to redis according to given event, increment and now(timestamp) value, It works with pipline.
func Push(event string, increment, now uint64) (interface{}, error) {

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

func Query(event string, start, end uint64, resolutionKey string) []TstampValTuple {
	redisConnection := *hnVars.redisConnection
	eventKey := getEventKey(event, resolutionKey)
	timestamps := getTimeStampsForQuery(start, end, resolutions[resolutionKey])
	values, err := redisConnection.Do("HMGET", eventKey, timestamps)
	fmt.Println(values)
	Check(err)
	tuple := make([]TstampValTuple, len(timestamps))

	for key, timestamp := range timestamps {
		tuple = append(tuple, TstampValTuple{x: timestamp, y: 0})
		//values.(string)[key]
		_, _ = values, key
	}

	return tuple
}

// GetCurrentTimeStamp returns current timestamp as type uint64
// !private
func GetCurrentTimeStamp() uint64 {
	return uint64(time.Now().Unix())
}

func GetSecFromRelativeTime(timespan string) uint64 {
	re := regexp.MustCompile(`^(\d+)\s(\w+)$`)
	match := re.FindStringSubmatch(timespan)
	if len(match) != 3 {
		panic("Configuration Err: timespan is not a valid value '" + timespan + "'")
	}
	re = regexp.MustCompile(`^` + match[2] + `*`)
	multiplier, _ := strconv.ParseUint(match[1], 10, 0)
	for key, resolution := range resolutions {
		if re.MatchString(key) {
			return resolution * multiplier
		}

	}
	return resolutions["day"] * multiplier
}

func GetResolutionKey(resolutionKey string) string {
	if resolutionKey != "" {
		return resolutionKey
	}
	return DEFAULT_RESOLUTION
}

func GetResolution(resolutionKey string) uint64 {
	resolution, ok := resolutions[resolutionKey]
	if ok {
		return resolution
	}
	return resolutions[DEFAULT_RESOLUTION]
}

// getTimeStampsForPush yields timestamps for each resolution which defined in resolutions hash map
// !private
func getTimeStampsForPush(now uint64) chan TimeResolution {
	if now == 0 {
		now = GetCurrentTimeStamp()
	}
	timeResolution := make(chan TimeResolution)
	go func() {
		defer close(timeResolution)
		for resolution, timestamp := range resolutions {
			timeResolution <- TimeResolution{resolution: resolution, timestamp: roundTime(now, timestamp)}
		}
	}()

	return timeResolution
}

func getTimeStampsForQuery(start, end, resolution uint64) []uint64 {
	return _range(roundTime(start, resolution), roundTime(end, resolution), resolution)
}

// getEventKey returns the event key according to given event name and resolution key
// !private
func getEventKey(event, resolutionKey string) string {
	return fmt.Sprintf("simmetrica:%s:%s", event, resolutionKey)
}

// roundTime rounds given time according to given resolution which defined in resolutions hash map
func roundTime(time, resolution uint64) uint64 {
	return uint64(time - (time % resolution))
}

// _range python like range, _func it is not idiomatic in go, need to seperate than built-in range.
func _range(start, end, frequency uint64) []uint64 {
	diff := start - end
	if diff <= 0 {
		return make([]uint64, 1)
	}
	//	fmt.Println("Start : ", start, "End : ", end)
	rangeSlice := make([]uint64, (0-diff)/frequency)
	current := start
	for current < end {
		current = current + frequency
		if current < end {
			rangeSlice = append(rangeSlice, current)
		}
		//		fmt.Println("Inc : ", start)
	}
	return rangeSlice
}
