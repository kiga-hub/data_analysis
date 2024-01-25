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

# 数据修复工具

提供两种模式
- 1.单个wav文件头部添加1个byte
- 2.并行处理原始数据，根据第一个正确文件，修复后面所有数据，以采集器ID为单位

## 模式1
```bash
./analysis -mode analysis  -path ./data -out ./rewrite
```

原始数据放在文件夹./data下，修复后的数据会放在/rewrite下。

生成结果会保存为json格式，modify_info.json

```

第一次修复数据完毕后生成modify_info.json

下一次修复数据开始会载入上次生成的modify_info.json,修复完毕后更新modify_info.json

## 模式2
```bash
./analysis -mode repair  -name 1.wav -out ./rewrite -head
```

原始数据放在./data下，指定修复后的数据路径./rewrite。 并置标志位head，指定该程序，只对wav文件头部添加1字节。

## 说明
```bash
Usage of ./repair:
  -head
        add head only
  -name string
        wav file name
  -out string
        audio or vibrate (default "H:/rewrite")
  -path string
        rawdata file path (default "H:/data")
  -type string
        audio or vibrate (default "audio")
```
