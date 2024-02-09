package config

import (
	"encoding/json"
	"hub/pkg/model"
	"hub/pkg/utils"
	"log"
	"time"

	"github.com/gomodule/redigo/redis"
)

var (
	redisPool = redis.Pool{
		MaxIdle:     9,
		MaxActive:   9,
		Wait:        true,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", "localhost:6379", redis.DialConnectTimeout(5*time.Second))
		},
	}
)

// servername, error, status code
// if status code = 0 -> either ready server or got an error
// if status code = 1 -> all the server are busy
func FindReadyServer(mapName string) (string, error, int) {
	conn := redisPool.Get()
	defer conn.Close()
	result, err := redis.Strings(conn.Do("HGETALL", mapName))
	if err != nil {
		log.Println("Error getting map:", err)
		return "", err, 0
	}

	// Parse the result into key-value pairs
	for i := 0; i < len(result); i += 2 {
		serverName := result[i]
		status := result[i+1]
		// log.Printf("ServerName: %s, Status: %s\n", serverName, status)
		if status == "ready" {
			return serverName, nil, 0
		}
	}
	return "", nil, 1
}

// in the hub
func SetInitialServerStatusInRedis(serverMap map[string]string, mapName string) error {
	conn := redisPool.Get()
	defer conn.Close()
	log.Printf("serverMap: %v\n", serverMap)
	for serverName := range serverMap {
		_, err := conn.Do("HSET", mapName, serverName, "ready")
		if err != nil {
			log.Printf("error setting initial status for %v: [%v]", serverName, err)
			return err
		}
	}
	return nil
}

// create a Map type redis for storing RequestId, CurrentCompletion, TotalChunks state
func SetInitialRequestStatus(requestId string, totalChunks int, currentCompletion int, timeStamp string) error {
	conn := redisPool.Get()
	defer conn.Close()

	redisRequestReportModel := model.RedisRequestReportModel{TotalChunks: totalChunks, CurrentCompleted: currentCompletion, TimeStamp: timeStamp}
	jsonModel, err := redisRequestReportModel.JSONMarshal()
	if err != nil {
		log.Printf("error marshalling redis report map: [%v]", err)
		return err
	}
	_, err = conn.Do("HSET", utils.REDIS_REQUEST_MAP_NAME, requestId, jsonModel)
	if err != nil {
		log.Printf("error setting initial status for Request %v: [%v]", requestId, err)
		return err
	}

	log.Printf("Successfully addition of RequestId: %v data in Redis", requestId)
	return nil
}

// fetch RequestId data and when TotalChunks == CurrentCompletion delete that Map
func FetchRequestStatus(requestId string, module_env string) (int, string) {
	conn := redisPool.Get()
	defer conn.Close()

	currentRemaining := -1
	innerJSON, err := redis.Bytes(conn.Do("HGET", utils.REDIS_REQUEST_MAP_NAME, requestId))
	if err != nil {
		log.Printf("error getting map: [%v]", err)
		return -1, ""
	}
	var retrievedInnerMap model.RedisRequestReportModel
	err = json.Unmarshal(innerJSON, &retrievedInnerMap)
	if err != nil {
		log.Printf("error unmarshalling json: [%v]", err)
		return -1, ""
	}
	currentCompleted := retrievedInnerMap.CurrentCompleted
	totalChunks := retrievedInnerMap.TotalChunks
	timeStamp := retrievedInnerMap.TimeStamp
	currentCompleted++
	log.Printf("CurrentCompleted: %d, TotalChunks: %d\n", currentCompleted, totalChunks)
	if currentCompleted == totalChunks {
		// remove this from redish
		err := DeleteRedisData(requestId, module_env)
		if err != nil {
			return -1, timeStamp
		} else {
			return 0, timeStamp
		}
	} else {
		// add back to redis
		SetInitialRequestStatus(requestId, totalChunks, currentCompleted, timeStamp)
	}
	currentRemaining = totalChunks - currentCompleted
	return currentRemaining, timeStamp

}

// Fetching Module Environment Tags
func FetchModuleEnvironment(module_env string) (model.RedisEnvironmentModuleList, error) {
	conn := redisPool.Get()
	defer conn.Close()

	var retrievedInnerMap model.RedisEnvironmentModuleList

	innerJSON, err := redis.Bytes(conn.Do("HGET", utils.REDIS_ENVIRONMENT_MAP_NAME, module_env))
	if err != nil {
		log.Printf("error getting map: [%v]", err)
		return retrievedInnerMap, err
	}
	err = json.Unmarshal(innerJSON, &retrievedInnerMap)
	if err != nil {
		log.Printf("error unmarshalling json: [%v]", err)
		return retrievedInnerMap, err
	}
	return retrievedInnerMap, nil
}

// Adding Request Module and Environment along with tags in Redis
func SettingModuleEnvironment(module_env string, requestId string, tags []string) error {
	conn := redisPool.Get()
	defer conn.Close()

	// firstly need to fetch previously data on which i have to add another request
	list, err := FetchModuleEnvironment(module_env)

	if err != nil {
		log.Printf("error marshalling redis report map in Fetching Module Environment Might not be present in the Redis: [%v]", err)
	}

	additionOfNewRequestTags := model.RedisEnvironmentModule{RequestId: requestId, Tags: tags}

	list.List_Tags = append(list.List_Tags, additionOfNewRequestTags)
	var model model.RedisEnvironmentModuleList
	model.List_Tags = list.List_Tags

	jsonModel, err := model.JSONMarshal()
	if err != nil {
		log.Printf("error marshalling redis report map in Module Environment: [%v]", err)
		return err
	}

	_, err = conn.Do("HSET", utils.REDIS_ENVIRONMENT_MAP_NAME, module_env, jsonModel)
	if err != nil {
		log.Printf("error setting initial Module ENvironment Data %v: [%v]", requestId, err)
		return err
	}

	log.Printf("Successfully addition of Module Environament data: %v data in Redis [%v]", requestId, list.List_Tags)
	return nil

}

// deleteing redis Data
func DeleteRedisData(requestId string, module_env string) error {
	conn := redisPool.Get()
	defer conn.Close()
	log.Printf("Map Deletions for RequestId : %v", requestId)
	_, er := conn.Do("HDEL", utils.REDIS_REQUEST_MAP_NAME, requestId)
	if er != nil {
		log.Printf("Error in Deletion of RequestId : %v", requestId)
		return er
	}
	log.Printf("Successful deletion of RequestId : %v", requestId)

	// firstly need to fetch previously data on which i have to add another request
	list, err := FetchModuleEnvironment(module_env)
	if err != nil {
		log.Printf("error marshalling redis report map in Fetching Module Environment Might not be present in the Redis: [%v]", err)
	}

	log.Printf("Size Before Removing Request from Environmental Module: %v", len(list.List_Tags))
	var newList model.RedisEnvironmentModuleList

	if len(list.List_Tags) == 1 {
		_, er := conn.Do("HDEL", utils.REDIS_ENVIRONMENT_MAP_NAME, module_env)
		if er != nil {
			log.Printf("Error in Deletion of RequestId : %v", requestId)
			return er
		}
		log.Printf("Successful deletion of Environment Module : %v", requestId)
		return nil
	}

	for i := 0; i < len(list.List_Tags); i++ {
		value := list.List_Tags[i]
		if value.RequestId == requestId {
			log.Printf("Removing requestId %v from Environment Module List", requestId)
		} else {
			newList.List_Tags = append(newList.List_Tags, value)
		}
	}

	var model model.RedisEnvironmentModuleList
	model.List_Tags = newList.List_Tags

	jsonModel, err := model.JSONMarshal()
	if err != nil {
		log.Printf("error marshalling redis report map in Module Environment: [%v]", err)
		return err
	}

	_, err = conn.Do("HSET", utils.REDIS_ENVIRONMENT_MAP_NAME, module_env, jsonModel)
	if err != nil {
		log.Printf("error setting initial Module ENvironment Data %v: [%v]", requestId, err)
		return err
	}

	log.Printf("Successfully deletion from Module Environament data: %v data in Redis %v", requestId, len(newList.List_Tags))
	return nil

}
