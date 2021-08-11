package util

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cast"
)

type Analysis struct {
	FileName string
	Value    int64
}

func StructToCsv(collectorid string, Data []Analysis) {
	newFile, err := os.Create(collectorid)
	if err != nil {
		fmt.Println(err)
	}
	defer func() {
		newFile.Close()
	}()
	// 写入UTF-8
	newFile.WriteString("\xEF\xBB\xBF") // 写入UTF-8 BOM，防止中文乱码
	// 写数据到csv文件
	w := csv.NewWriter(newFile)
	header := []string{"FileName", "Value"} //标题

	w.Write(header)
	fmt.Println(Data)
	for _, v := range Data {
		context := []string{
			v.FileName,
			cast.ToString(v.Value),
		}
		w.Write(context)
	}

	if err := w.Error(); err != nil {
		log.Fatalln("error writing csv:", err)
	}
	w.Flush()
}
