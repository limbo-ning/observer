package resource

import (
	"bytes"
	"errors"
	"image"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"os"
	"strings"

	"obsessiontech/common/util"
)

type File struct {
	Name    string                 `json:"name"`
	Path    string                 `json:"path"`
	Size    int                    `json:"size"`
	IsDir   bool                   `json:"isDir"`
	ModTime util.Time              `json:"modTime"`
	Extra   map[string]interface{} `json:"extra"`
}

func isImage(fileName string) bool {

	parts := strings.Split(fileName, ".")
	if len(parts[0]) == 1 {
		log.Println("error check image type: ", fileName)
		return false
	}

	fileType := parts[len(parts)-1]
	for _, permits := range Config.PermitFileType {
		if permits.Name == "image" {
			for _, subfix := range permits.Subfix {
				if subfix.Name == fileType {
					return true
				}
			}
		}
	}

	return false
}

func validateFileType(fileType, filename string) error {
	fileType = strings.ToLower(fileType)
	filename = strings.ToLower(filename)
	for _, permits := range Config.PermitFileType {
		if permits.Name == fileType {
			for _, subfix := range permits.Subfix {
				if strings.HasSuffix(filename, subfix.Name) {
					return nil
				}
			}
		}
	}
	return errors.New("不支持的文件类型")
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	}
	return true
}

func UploadResource(siteID string, files map[string][]*multipart.FileHeader, params map[string][]string) ([]string, error) {

	result := make([]string, 0)

	for fileType, fileHeaders := range files {

		var paths []string
		if _, exists := params[fileType]; exists {
			paths = params[fileType]
		} else if _, exists := params[fileType+"_path"]; exists {
			paths = params[fileType+"_path"]
		}
		if paths == nil || len(paths) != len(fileHeaders) {
			log.Println("error file param not match: ", fileType, paths, fileHeaders, files, params)
			return nil, errors.New("file param not match")
		}

		for i, fh := range fileHeaders {
			if err := validateFileType(fileType, fh.Filename); err != nil {
				return nil, err
			}

			path := paths[i]
			if path == "" {
				return nil, errors.New("file path empty")
			}

			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}

			if !strings.HasSuffix(path, "/") {
				path += "/"
			}

			var uploadReader io.Reader

			tmplFile, err := fh.Open()
			if err != nil {
				return nil, err
			}

			uploadReader = tmplFile
			if isImage(fh.Filename) {
				b, err := ioutil.ReadAll(tmplFile)
				if err != nil {
					return nil, err
				}
				imageReader := bytes.NewReader(b)
				if _, _, err = image.DecodeConfig(imageReader); err != nil {
					log.Println("error file is not image: ", err)
					return nil, errors.New("图片无法识别")
				}
				uploadReader = bytes.NewReader(b)
			}

			if !isExist(Config.ResourceFolderPath + siteID + path) {
				if err := os.MkdirAll(Config.ResourceFolderPath+siteID+path, os.ModePerm); err != nil {
					return nil, err
				}
			}

			filename := fh.Filename
			renames, exists := params[fileType+"_rename"]
			if exists && len(renames) == len(fileHeaders) {
				filename = renames[i]
			}

			dest, err := os.Create(Config.ResourceFolderPath + siteID + path + filename)
			if err != nil {
				return nil, err
			}
			defer dest.Close()

			if _, err := io.Copy(dest, uploadReader); err != nil {
				return nil, err
			}

			result = append(result, path+filename)
		}
	}

	return result, nil
}

func MoveResource(siteID, filePath, destFilePath string) (e error) {
	if filePath == "" {
		return errors.New("filePath empty")
	}
	if filePath == destFilePath {
		return nil
	}
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}
	if !strings.HasPrefix(destFilePath, "/") {
		destFilePath = "/" + destFilePath
	}
	destFolder := destFilePath[:strings.LastIndex(destFilePath, "/")]

	if isExist(Config.ResourceFolderPath + siteID + filePath) {
		if destFilePath == "" {
			return errors.New("destFilePath empty")
		}

		defer func() {
			if e == nil {
				e = os.Remove(Config.ResourceFolderPath + siteID + filePath)
			}
		}()

		src, err := os.Open(Config.ResourceFolderPath + siteID + filePath)
		if err != nil {
			return err
		}
		defer src.Close()

		if !isExist(Config.ResourceFolderPath + siteID + destFolder) {
			if err := os.MkdirAll(Config.ResourceFolderPath+siteID+destFolder, os.ModePerm); err != nil {
				return err
			}
		}

		dest, err := os.Create(Config.ResourceFolderPath + siteID + destFilePath)
		if err != nil {
			return err
		}
		defer dest.Close()

		if _, err := io.Copy(dest, src); err != nil {
			return err
		}

		return nil
	}

	return errors.New("file not exists")
}

func DeleteResource(siteID, filePath string) error {
	if filePath == "" {
		return errors.New("filePath empty")
	}
	if isExist(Config.ResourceFolderPath + siteID + "/" + filePath) {
		return os.RemoveAll(Config.ResourceFolderPath + siteID + "/" + filePath)
	}

	return errors.New("file not exists")
}

func ListResource(siteID, folder string) ([]*File, error) {
	files, err := ioutil.ReadDir(Config.ResourceFolderPath + siteID + "/" + folder)
	if err != nil {
		return nil, err
	}

	result := make([]*File, 0)
	for _, f := range files {
		ret := &File{
			Name:    f.Name(),
			Size:    int(f.Size()),
			Path:    folder,
			IsDir:   f.IsDir(),
			ModTime: util.Time(f.ModTime()),
			Extra:   make(map[string]interface{}),
		}

		if isImage(f.Name()) {

			if !strings.HasSuffix(folder, "/") {
				folder += "/"
			}

			file, err := os.Open(Config.ResourceFolderPath + siteID + "/" + folder + f.Name())
			if err != nil {
				log.Println("error read file: ", err)
				continue
			}

			config, format, err := image.DecodeConfig(file)
			if err != nil {
				log.Println("error file is not image: ", f.Name(), err)
				continue
			}

			ret.Extra["format"] = format
			ret.Extra["width"] = config.Width
			ret.Extra["height"] = config.Height
			ret.Extra["isImage"] = true
		}

		result = append(result, ret)
	}

	return result, nil
}
