# 数据分析工具

- 并行处理修复后的数据(以数据正确为前提)，计算每个文件前1000字节平方和
- 计算后以每个采集器为单位生成csv文件，保存要分析的文件名和平方和

```golang
// type Analysis struct {
// 	FileName string
// 	Value    int64
// }

./analysis  -path ./data -out ./csv
```


## 说明
```bash
Usage of ./analysis:
  -out string
        audio or vibrate (default "H:/rewrite")
  -path string
        rawdata file path (default "H:/data")
  -type string
        audio or vibrate (default "audio")
```
