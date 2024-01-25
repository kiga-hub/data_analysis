package util

import (
	"errors"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
)

// TestFolderWritable stat of folder
func TestFolderWritable(folder string, logger *zap.SugaredLogger) (err error) {
	fileInfo, err := os.Stat(folder)
	if err != nil {
		return err
	}
	if !fileInfo.IsDir() {
		return errors.New("not a valid folder")
	}
	perm := fileInfo.Mode().Perm()
	logger.Debug("Folder", folder, "Permission:", perm)
	if 0200&perm != 0 {
		return nil
	}
	return errors.New("not writable")
}

// GetFileSize get file size
func GetFileSize(file *os.File) (size int64, err error) {
	var fi os.FileInfo
	if fi, err = file.Stat(); err == nil {
		size = fi.Size()
	}
	return
}

// RemoveFolders -
func RemoveFolders(dir string) error {
	exist, err := PathExists(dir)
	if err != nil {
		return err
	}
	if exist && strings.Contains(dir, ".ldb") {
		err := os.RemoveAll(dir)
		if err != nil {
			return err
		}
	}
	return nil
}

// FileExists  check file is exist
func FileExists(filename string) bool {

	_, err := os.Stat(filename)

	return !os.IsNotExist(err)

	// if os.IsNotExist(err) {
	// 	return false
	// }
	// return true

}

// PathExists -
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// CheckFile checkfile
func CheckFile(filename string) (exists, canRead, canWrite bool, modTime time.Time, fileSize int64) {
	exists = true
	fi, err := os.Stat(filename)
	if os.IsNotExist(err) {
		exists = false
		return
	}
	if fi.Mode()&0400 != 0 {
		canRead = true
	}
	if fi.Mode()&0200 != 0 {
		canWrite = true
	}
	modTime = fi.ModTime()
	fileSize = fi.Size()
	return
}

// GetFileList -
func GetFileList(path string, audiofiles map[string][]string, key string) {
	fs, _ := os.ReadDir(path)
	for _, file := range fs {
		if file.IsDir() {
			key = path + file.Name()
			GetFileList(path+file.Name(), audiofiles, key)
		} else {
			audiofiles[string(key)] = append(audiofiles[string(key)], path+"/"+file.Name())
		}
	}
}

// GetFileListName -
func GetFileListName(path string, audiofiles map[string]string, key string) {
	fs, _ := os.ReadDir(path)
	for _, file := range fs {
		if file.IsDir() {
			//	key += file.Name()[:2]
			GetFileListName(path+file.Name(), audiofiles, key) // key+file.Name()[:17]
		} else {
			if len(file.Name()) > 17 {
				audiofiles[string(key+file.Name()[:17])] = path + "/" + file.Name()
			}
		}
	}
}
