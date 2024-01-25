package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	utilpath "path/filepath"
	"strings"
	"sync"

	"analysis/util"

	"github.com/spf13/cast"
)

var (
	mode        string
	name        string
	rawdatapath string
	datatype    string
	rewritepath string
	head        bool
)

type AnalysisWork struct {
	decodeJobChans []chan PathStructure
	wg             *sync.WaitGroup
}

type output interface {
	io.Writer
	io.Closer
}

type Writer struct {
	output
	sampleBuf *bufio.Writer
}

func NewPCMWriter(out output) (wr *Writer) {
	wr = &Writer{}
	wr.output = out
	wr.sampleBuf = bufio.NewWriter(out)
	return
}

type RepairWork struct {
	decodeJobChans []chan PathStructure
	isWrong        *sync.Map
	wg             *sync.WaitGroup
	record         *sync.Map
	isFirst        *sync.Map
}

func (rp *RepairWork) ConcurrentProcessing(threadindex int, collectorid string, inputpath []string, outputpath, datatype string) {

	ps := &PathStructure{
		id:       collectorid,
		input:    inputpath,
		output:   outputpath,
		datatype: datatype,
	}

	rp.decodeJobChans[threadindex] <- *ps
}

func (rp *RepairWork) RepairData(input chan PathStructure) {
	for ps := range input {
		fmt.Println("It is First time Processing thread!")
		var isFirst bool
		var isWrong bool
		var record byte
		if isf, ok := rp.isFirst.LoadOrStore(ps.id, true); ok {
			isFirst = isf.(bool)
		}
		if isw, ok := rp.isWrong.Load(ps.id); ok {
			isWrong = isw.(bool)
		}
		if re, ok := rp.record.Load(ps.id); ok {
			record = re.(byte)
		}

		for index, filepath := range ps.input {
			if len(filepath) <= 93 {
				continue
			}
			s := len(filepath) - 93
			dstname := filepath[s : s+31] //94C96000C25A/20210526/09/audio/

			dstfilename := utilpath.Base(filepath)
			var buffer bytes.Buffer

			reader, err := util.NewReader(filepath)
			if err != nil {
				fmt.Println(err)
			}

			readbuf := make([]byte, reader.DataChunk.Size)
			n, err := reader.Read(readbuf)
			if n < 1 || err != nil {
				fmt.Printf("reader.Read %d, %v\n", reader.DataChunk.Size, err)
			}

			if isFirst {
				if Odd_Number(reader.DataChunk.Size) {
					record = readbuf[len(readbuf)-1:][0]   //保存最后1字节,准备移动至下一个文件
					buffer.Write(readbuf[:len(readbuf)-1]) //写入剩余数据
					isWrong = true                         // 置标志位为true，确认下一个文件需要操作
					isFirst = false                        // 以后的文件为需要偏移的数据
				} else {
					buffer.Write(readbuf) // 直接写入文件正确
					isWrong = false       // 置标志位为true，确认下一个文件不需要操作
					isFirst = false       // 以后的文件为需要偏移的数据
				}
			} else { // 非第一个创建出来的文件
				// 文件大小为奇数
				if Odd_Number(reader.DataChunk.Size) {
					if isWrong { //文件错误操作
						buffer.Write([]byte{record}) //获取上一个文件尾部1字节,写入1字节
						buffer.Write(readbuf)
						isWrong = false //置标志位，确认下一个文件需要操作
					} else {
						record = readbuf[len(readbuf)-1:][0]   //保存要操作的1字节-移动末尾1字节，至下一个文件
						buffer.Write(readbuf[:len(readbuf)-1]) // 写入偏移后的数据
						isWrong = true                         // 置标志位，确认下一个文件需要操作
					}
				}
				// 文件大小为偶数
				if !Odd_Number(reader.DataChunk.Size) {
					if isWrong { //文件错误 使用上一个文件尾部1字节
						buffer.Write([]byte{record})           //获取上一个文件尾部1字节，写入1字节
						buffer.Write(readbuf[:len(readbuf)-1]) //写入去除末尾1字节后的数据
						record = readbuf[len(readbuf)-1:][0]   //保存要操作的1字节
						isWrong = true                         //置标志位，确认下一个文件需要操作
					} else {
						buffer.Write(readbuf) // 直接写入文件正确
						isWrong = false       // 置标志位为true，确认下一个文件不需要操作
					}
				}
			}

			wavheader, _ := util.ConvertPCMToWavHeader(
				len(buffer.Bytes()),
				1,
				int(reader.FmtChunk.Data.SamplesPerSec),
				int(reader.FmtChunk.Data.BitsPerSamples),
			)

			err = Write2File(ps.output+"/"+dstname, ps.output+"/"+dstname+dstfilename, wavheader, buffer.Bytes())
			if err != nil {
				fmt.Println(err)
			}
			if index%17 == 0 {
				fmt.Println("output path: ", ps.output+"/"+dstname+dstfilename, reader.DataChunk.Size)
			}
		}

		data := &util.Data{
			CollectorID: ps.id,
			IsWrong:     isWrong,
			Byte:        record,
		}
		util.Write2json(*data)

		rp.wg.Done()
	}
}

func (rp *RepairWork) listFiles(dirname string, level int, fileapth map[string][]string, datatype string) {
	//level record current recursive level
	s := "|--"
	for i := 0; i < level; i++ {
		s = "|   " + s
	}
	fileInfos, err := ioutil.ReadDir(dirname)
	if err != nil {
		fmt.Println(err)
	}
	for _, fi := range fileInfos {
		filename := dirname + "/" + fi.Name()
		// fmt.Printf("%s%s\n", s, filename)
		if fi.IsDir() {
			//继续遍历fi这个目录
			rp.listFiles(filename, level+1, fileapth, datatype)
		}

		if level == 4 && !strings.Contains(filename, ".gz") && strings.Contains(filename, datatype) {
			///94C96000C25B/20210525/08/audio/20210525081426226_20210525081428657_32000_010012_01_E00100.wav
			map_key := filename[len(filename)-93 : len(filename)-81]
			fileapth[map_key] = append(fileapth[map_key], filename)
		}
	}
}

func Odd_Number(len uint32) bool {
	return len%2 == 1
}

type PathStructure struct {
	id       string
	input    []string
	output   string
	datatype string
}

// ./analysis(repair)  -path ./data -out ./rewrite -type audio
func init() {
	flag.StringVar(&mode, "mode", "", "analysis or repaire")
	flag.StringVar(&name, "name", "", "wav file name")
	flag.BoolVar(&head, "head", false, "add head only")
	flag.StringVar(&datatype, "type", "audio", "audio or vibrate")
	flag.StringVar(&rewritepath, "out", "H:/rewrite", "audio or vibrate")
	flag.StringVar(&rawdatapath, "path", "H:/data", "rawdata file path")
}

func (rp *AnalysisWork) ConcurrentProcessing(threadindex int, collectorid string, inputpath []string, outputpath, datatype string) {

	ps := &PathStructure{
		id:       collectorid,
		input:    inputpath,
		output:   outputpath,
		datatype: datatype,
	}

	rp.decodeJobChans[threadindex] <- *ps
}

func (rp *AnalysisWork) listFiles(dirname string, level int, fileapth map[string][]string, datatype string) {
	//level record current recursive level
	s := "|--"
	for i := 0; i < level; i++ {
		s = "|   " + s
	}
	fileInfos, err := ioutil.ReadDir(dirname)
	if err != nil {
		fmt.Println(err)
	}
	for _, fi := range fileInfos {
		filename := dirname + "/" + fi.Name()
		// fmt.Printf("%s%s\n", s, filename)
		if fi.IsDir() {
			//继续遍历fi这个目录
			rp.listFiles(filename, level+1, fileapth, datatype)
		}

		if level == 4 && !strings.Contains(filename, ".gz") && strings.Contains(filename, datatype) {
			///94C96000C25B/20210525/08/audio/20210525081426226_20210525081428657_32000_010012_01_E00100.wav
			map_key := filename[len(filename)-93 : len(filename)-81]
			fileapth[map_key] = append(fileapth[map_key], filename)
		}
	}
}

func (rp *AnalysisWork) AnalysisData(input chan PathStructure) {
	for ps := range input {
		fmt.Println("It is First time Processing thread!")

		analysis := []util.Analysis{} //make([]util.Analysis, len(ps.input))

		for index, filepath := range ps.input {
			if len(filepath) <= 93 {
				continue
			}
			s := len(filepath) - 93
			dstname := filepath[s : s+31] //94C96000C25A/20210526/09/audio/

			dstfilename := utilpath.Base(filepath)
			reader, err := util.NewReader(filepath)
			if err != nil {
				fmt.Println(err)
			}

			readbuf := make([]byte, reader.DataChunk.Size)
			n, err := reader.Read(readbuf)
			if n < 1 || err != nil {
				fmt.Printf("reader.Read %d, %v\n", reader.DataChunk.Size, err)
			}

			if reader.DataChunk.Size > 1000 {
				value := util.GetQuadratic(readbuf[:1000])
				if err != nil {
					fmt.Println(err)
					continue
				}
				analysis = append(analysis, util.Analysis{
					FileName: filepath,
					Value:    cast.ToInt64(value),
				})
			}

			if index%17 == 0 {
				fmt.Println("output path: ", ps.output+"/"+dstname+dstfilename, reader.DataChunk.Size)
			}
		}

		util.StructToCsv(ps.output+"/"+ps.id+".csv", analysis)
		rp.wg.Done()
	}
}

func AnalysisDatas() {
	rp := &AnalysisWork{
		wg: new(sync.WaitGroup),
	}

	filepathmap := map[string][]string{}
	rp.listFiles(rawdatapath, 0, filepathmap, datatype)

	rp.decodeJobChans = make([]chan PathStructure, len(filepathmap))

	for i := 0; i < len(filepathmap); i++ {
		rp.decodeJobChans[i] = make(chan PathStructure, 1024)
		go rp.AnalysisData(rp.decodeJobChans[i])
	}

	err := os.Mkdir(rewritepath, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

	index := 0

	for k, v := range filepathmap {
		rp.wg.Add(1)
		rp.ConcurrentProcessing(index, k, v, rewritepath, datatype)
		index++
	}

	rp.wg.Wait()

	fmt.Println("++++>>>>all gorutine finish...<<<<++++")
}

func AddHead() {
	filepath := "./data/" + name

	reader, err := util.NewReader(filepath)
	if err != nil {
		fmt.Println(err)
	}
	// fmt.Println(reader.DataChunk.Size)
	readbuf := make([]byte, reader.DataChunk.Size)
	n, err := reader.Read(readbuf)
	if n < 1 || err != nil {
		fmt.Printf("reader.Read %d, %v\n", reader.DataChunk.Size, err)
	}

	var buffer bytes.Buffer

	if Odd_Number(reader.DataChunk.Size) {
		buffer.Write(readbuf[1:2])
		buffer.Write(readbuf)
	} else {
		buffer.Write(readbuf[1:2])
		buffer.Write(readbuf)
		buffer.Write(readbuf[len(readbuf)-1:])
	}

	wavheader, err := util.ConvertPCMToWavHeader(
		len(buffer.Bytes()),
		1,
		int(reader.FmtChunk.Data.SamplesPerSec),
		int(reader.FmtChunk.Data.BitsPerSamples),
	)
	if err != nil {
		fmt.Println(err)
	}

	err = Write2File(rewritepath, rewritepath+"/"+name[:len(name)-4]+"_"+name, wavheader, buffer.Bytes())
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("++++>>>>add head is Done...<<<<++++")
}

func RepairDatas() {
	rp := &RepairWork{
		isWrong: &sync.Map{},
		wg:      new(sync.WaitGroup),
		record:  &sync.Map{},
		isFirst: &sync.Map{},
	}

	filepathmap := map[string][]string{}
	rp.listFiles(rawdatapath, 0, filepathmap, datatype)

	rp.decodeJobChans = make([]chan PathStructure, len(filepathmap))

	for i := 0; i < len(filepathmap); i++ {
		rp.decodeJobChans[i] = make(chan PathStructure, 1024)
		go rp.RepairData(rp.decodeJobChans[i])
	}

	err := os.Mkdir(rewritepath, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

	index := 0

	for k, v := range filepathmap {
		var b byte
		err, data := util.Readjson(k)
		if err != nil {
			rp.record.LoadOrStore(k, b)
			rp.isWrong.LoadOrStore(k, false)
			rp.isFirst.LoadOrStore(k, true)
		} else {
			rp.record.LoadOrStore(k, data.Byte)
			rp.isWrong.LoadOrStore(k, data.IsWrong)
			rp.isFirst.LoadOrStore(k, false)
		}

		rp.wg.Add(1)
		rp.ConcurrentProcessing(index, k, v, rewritepath, datatype)
		index++
	}

	rp.wg.Wait()

	fmt.Println("++++>>>>all gorutine finish...<<<<++++")
}

func Write2File(subdirectory, outputFileName string, header, data []byte) (err error) {
	err = os.MkdirAll(subdirectory, os.ModePerm)
	if err != nil {
		return fmt.Errorf("os.MkdirAll %s, %v", subdirectory, err)
	}

	outputFile, err := os.Create(outputFileName)
	if err != nil {
		return fmt.Errorf("exists %v", err)
	}
	defer outputFile.Close()

	writer := NewPCMWriter(outputFile)
	_, err = writer.Write(header)
	if err != nil {
		return fmt.Errorf("write header %v", err)
	}

	_, err = writer.Write(data)
	if err != nil {
		return fmt.Errorf("write data %v", err)
	}
	defer writer.Close()
	if err != nil {
		return fmt.Errorf("write Close %v", err)
	}

	return nil
}

func main() {
	flag.Parse()
	if mode == "analysis" {
		AnalysisDatas()
	}
	if mode == "repair" {
		if !head {
			RepairDatas()
		} else {
			AddHead()
		}
	}
}
