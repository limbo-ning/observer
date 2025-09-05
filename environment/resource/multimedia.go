package resource

import (
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

func extractFileFormat(fileName string) string {
	lastIndex := strings.LastIndex(fileName, ".")
	if lastIndex < 0 {
		return ""
	}

	return fileName[lastIndex+1:]
}

func convertMultimedia(input []byte, fileName, toFormat string) ([]byte, string, error) {

	format := extractFileFormat(fileName)
	if format == "" {
		return nil, "", errors.New("无法识别文件类型")
	}

	if err := ioutil.WriteFile(os.TempDir()+"/"+fileName, input, os.ModePerm); err != nil {
		return nil, "", err
	}
	defer os.Remove(os.TempDir() + "/" + fileName)

	toFileName := strings.Replace(fileName, format, toFormat, 1)

	cmd := exec.Command("ffmpeg", "-i", os.TempDir()+"/"+fileName, os.TempDir()+"/"+toFileName)
	err := cmd.Start()
	if err != nil {
		return nil, "", err
	}
	err = cmd.Wait()
	if err != nil {
		return nil, "", err
	}
	defer os.Remove(os.TempDir() + "/" + toFileName)

	converted, err := ioutil.ReadFile(os.TempDir() + "/" + toFileName)
	if err != nil {
		return nil, "", err
	}

	return converted, toFileName, nil
}
