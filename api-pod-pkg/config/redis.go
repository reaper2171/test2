package config

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"bitbucket.com/testing-cypress-server/server1/pkg/model"
)

// import (
// 	"bytes"
// 	"encoding/json"
// 	"errors"
// 	"log"
// 	"net/http"

// 	"bitbucket.com/testing-cypress-server/server1/pkg/model"
// 	"bitbucket.com/testing-cypress-server/server1/pkg/utils"

// 	"github.com/gomodule/redigo/redis"
// )

func UpdateRedis(podRedisModel model.PodRedisModel) error {
	jsonMarshall, err := json.Marshal(podRedisModel)

	if err != nil {
		log.Printf("error in pod redis request: %v", err)
		return err
	}

	resp, err := http.Post("http://192.168.6.149/updateRedisFromPod", "application/json", bytes.NewBuffer(jsonMarshall))

	if err != nil {
		log.Printf("error in POST request to %v: [%v]", podRedisModel.ServerName, err)
	}

	defer resp.Body.Close()

	return err
}

// // in the server
// func SetServerAsBusy(serverName, mapName string) error {
// 	conn := redisPool.Get()
// 	defer conn.Close()

// 	_, err := conn.Do("HSET", mapName, serverName, "busy")
// 	if err != nil {
// 		log.Printf("error updating status in redis: [%v]", err)
// 		return err
// 	}
// 	log.Printf("successfully updated %s as busy", serverName)
// 	return nil
// }

// // in the server
// func SetServerAsFree(conn redis.Conn, serverName, mapName string) error {
// 	_, err := conn.Do("HSET", mapName, serverName, "ready")
// 	if err != nil {
// 		log.Printf("error updating status in redis: [%v]", err)
// 		return err
// 	}
// 	log.Printf("successfully updated %s as free", serverName)
// 	return nil
// }

// func InitServerReportMap(serverName string) error {
// 	serverReport := model.ServerReport{TotalTest: 0, TestPassed: 0, TestFailed: 0}
// 	json, err := serverReport.JsonMarshal()
// 	if err != nil {
// 		log.Printf("error marshalling json: [%v]", err)
// 		return err
// 	}

// 	conn := redisPool.Get()
// 	defer conn.Close()

// 	_, err = conn.Do("HSET", utils.REDIS_TEST_RESULTS, serverName, json)
// 	if err != nil {
// 		log.Printf("error setting nested map field: [%v]", err)
// 		return err
// 	}
// 	log.Printf("server report map initialized")
// 	return nil
// }

// func UpdateServerReport(conn redis.Conn, serverName string, passed bool) error {
// 	serverReport, err := GetServerReport(serverName)
// 	if err != nil {
// 		log.Printf("error getting server report: [%v]", err)
// 		return err
// 	}

// 	if passed {
// 		serverReport.TestPassed += 1
// 	} else {
// 		serverReport.TestFailed += 1
// 	}
// 	serverReport.TotalTest += 1
// 	serverReportMarshalled, err := serverReport.JsonMarshal()
// 	if err != nil {
// 		log.Printf("error marshalling json: [%v]", err)
// 		return err
// 	}
// 	_, err = conn.Do("HSET", utils.REDIS_TEST_RESULTS, serverName, serverReportMarshalled)
// 	if err != nil {
// 		log.Printf("error updating server report: [%v]", err)
// 		return err
// 	}
// 	return nil
// }

// func GetServerReportWithKey(serverName, key string) (int, error) {
// 	conn := redisPool.Get()
// 	defer conn.Close()
// 	innerJSON, err := redis.Bytes(conn.Do("HGET", utils.REDIS_TEST_RESULTS, serverName))
// 	if err != nil {
// 		log.Printf("error getting map: [%v]", err)
// 		return -1, err
// 	}
// 	var retrievedInnerMap model.ServerReport
// 	err = json.Unmarshal(innerJSON, &retrievedInnerMap)
// 	if err != nil {
// 		log.Printf("error unmarshalling json: [%v]", err)
// 		return -1, err
// 	}
// 	if key == "test_passed" {
// 		return retrievedInnerMap.TestPassed, nil
// 	} else if key == "test_failed" {
// 		return retrievedInnerMap.TestFailed, nil
// 	} else if key == "total_tests" {
// 		return retrievedInnerMap.TotalTest, nil
// 	} else {
// 		return -1, errors.New("key doesn't exist in the map")
// 	}
// }

// func GetServerReport(serverName string) (model.ServerReport, error) {
// 	conn := redisPool.Get()
// 	defer conn.Close()
// 	innerJSON, err := redis.Bytes(conn.Do("HGET", utils.REDIS_TEST_RESULTS, serverName))
// 	if err != nil {
// 		log.Printf("error getting map: [%v]", err)
// 		return model.ServerReport{}, err
// 	}
// 	var retrievedInnerMap model.ServerReport
// 	err = json.Unmarshal(innerJSON, &retrievedInnerMap)
// 	if err != nil {
// 		log.Printf("error unmarshalling json: [%v]", err)
// 		return model.ServerReport{}, err
// 	}
// 	return retrievedInnerMap, nil
// }

// func UpdateServer(serverName string, passed bool) error {
// 	conn := redisPool.Get()
// 	defer conn.Close()

// 	err := SetServerAsFree(conn, serverName, utils.REDIS_MAP_NAME)
// 	if err != nil {
// 		log.Printf("error setting %v as free", serverName)
// 		return err
// 	}
// 	err = UpdateServerReport(conn, serverName, passed)
// 	if err != nil {
// 		log.Printf("error upating %v report map", serverName)
// 		return err
// 	}
// 	return nil
// }

// // func SetTotalRuns(val, mapName, serverName string) error {
// // 	conn := redisPool.Get()
// // 	defer conn.Close()
// // 	_, err := conn.Do("HSET", mapName, serverName, val)
// // 	if err != nil {
// // 		log.Printf("error initializing total runs: [%v]", err)
// // 		return err
// // 	}
// // 	return nil
// // }

// // func GetTotalTestRuns(mapName, serverName string) (string, error) {
// // 	conn := redisPool.Get()
// // 	defer conn.Close()

// // 	val, err := redis.String(conn.Do("HGET", mapName, serverName))
// // 	if err != nil {
// // 		log.Printf("can't get total runs for: [%v]", serverName)
// // 		return "", err
// // 	}
// // 	return val, nil

// // }

// // func UpdateTotalTestRuns(serverName string) error {
// // 	conn := redisPool.Get()
// // 	defer conn.Close()
// // 	val, err := GetTotalTestRuns(utils.REDIS_TEST_RESULTS, serverName)
// // 	if err != nil {
// // 		log.Printf("error getting total runs: [%v]", err)
// // 		return err
// // 	}
// // 	prevVal, err := strconv.Atoi(val)
// // 	if err != nil {
// // 		log.Printf("error converting string to int")
// // 		return err
// // 	}
// // 	err = SetTotalRuns(strconv.Itoa(prevVal+1), utils.REDIS_TEST_RESULTS, serverName)
// // 	if err != nil {
// // 		log.Printf("error setting total runs: [%v]", err)
// // 	}
// // 	return nil
// // }
