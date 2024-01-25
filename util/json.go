package util

import (
	"encoding/json"
	"fmt"
	"os"
)

type Data struct {
	CollectorID string `json:"CollectorID"`
	IsWrong     bool   `json:"IsWrong"`
	Byte        byte   `json:"Byte"`
}

// 创建结构体
type Modify struct {
	Platform string `json:"Platform"`
	Address  string `json:"Address"`
	Data     []Data `json:"Data"`
	Num      int    `json:"Num"`
}

// Readjson Read content from json file
func Readjson(collectorid string) (error, Data) {

	var modify Modify
	bytes, err := os.ReadFile("modify_info.json")
	if err != nil {
		return fmt.Errorf("err %v", err), Data{}
	}

	err = json.Unmarshal(bytes, &modify)
	if err != nil {
		return fmt.Errorf("err %v", err), Data{}
	}

	index, _, isContain := IsContainData(modify, collectorid)
	if !isContain {
		return nil, Data{}
		//fmt.Println(modify.Data[index].CollectorID, modify.Data[index].IsWrong, modify.Data[index].Byte)
	}
	data := &Data{
		modify.Data[index].CollectorID,
		modify.Data[index].IsWrong,
		modify.Data[index].Byte,
	}
	return nil, *data
}

// Write2json Write content to json file
func Write2json(data Data) error {

	var modify Modify

	bytes, err := os.ReadFile("modify_info.json")
	if err != nil {
		filePtr, err := os.Create("modify_info.json")
		if err != nil {
			fmt.Println("Create file failed", err.Error())
			return err
		}
		defer filePtr.Close()

		modify.Data = append(modify.Data, data)
		modify.Platform = "Node"
		modify.Address = "Location"
		modify.Num = 1

		content, err := json.MarshalIndent(modify, "", "  ")
		if err != nil {
			return err
		}

		filePtr.Write(content)
		return nil
	}

	err = json.Unmarshal(bytes, &modify)
	if err != nil {
		return err
	}

	index, num, isContain := IsContainData(modify, data.CollectorID)
	if !isContain {
		modify.Data = append(modify.Data, data)
		num++
	} else {
		modify.Data[index].IsWrong = data.IsWrong
		modify.Data[index].Byte = data.Byte
	}
	modify.Num = num

	result, err := json.MarshalIndent(modify, "", "    ")
	if err != nil {
		return err
	}

	if err := os.WriteFile("modify_info.json", result, 0644); err != nil {
		return err
	}

	return nil
}

// IsContainData check json file is contain collectorid or not
func IsContainData(modify Modify, collectorid string) (int, int, bool) {
	num := 0
	isContain := false
	index := 0
	for k, v := range modify.Data {
		if v.CollectorID == collectorid {
			isContain = true
			index = k
		}
		num++
	}
	return index, num, isContain
}
