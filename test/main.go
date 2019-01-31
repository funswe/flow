package main

import (
	"fmt"
	"encoding/json"
)

type appInfo struct {
	Appid string `json:"appId"`
}

type response struct {
	RespCode string  `json:"respCode"`
	RespMsg  string  `json:"respMsg"`
	TestCode int     `json:"testCode,string"`
	AppInfo  appInfo `json:"app"`
}

type JsonResult struct {
	Resp response `json:"resp"`
}

func main() {
	jsonstr := `{"resp": {"testCode": "123","respCode": "000000","respMsg": "成功","app": {"appId": "d12abd3da59d47e6bf13893ec43730b8"}}}`
	var JsonRes JsonResult
	json.Unmarshal([]byte(jsonstr), &JsonRes)
	fmt.Println("after parse", JsonRes.Resp.AppInfo, JsonRes.Resp.RespCode, JsonRes.Resp.TestCode, JsonRes.Resp.RespMsg)
}
