package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	utilpath "path/filepath"
	"strings"
	"sync"

	"analysis/util"

	"github.com/spf13/cast"
)

type RepairWork struct {
	decodeJobChans []chan PathStructure
	wg             *sync.WaitGroup
}

type PathStructure struct {
	id       string
	input    []string
	output   string
	datatype string
}

var (
	rawdatapath string
	datatype    string
	rewritepath string
)

//./repair  -path ./data -out ./rewrite -type audio
func init() {
	flag.StringVar(&datatype, "type", "audio", "audio or vibrate")
	flag.StringVar(&rewritepath, "out", "H:/rewrite", "audio or vibrate")
	flag.StringVar(&rawdatapath, "path", "H:/data", "rawdata file path")
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

func main() {
	flag.Parse()
	RepairDatas()
}

func RepairDatas() {
	rp := &RepairWork{
		wg: new(sync.WaitGroup),
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
		rp.wg.Add(1)
		rp.ConcurrentProcessing(index, k, v, rewritepath, datatype)
		index++
	}

	rp.wg.Wait()

	fmt.Println("++++>>>>all gorutine finish...<<<<++++")
}
