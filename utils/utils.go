package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/projecteru/eru-agent/common"
	"github.com/projecteru/eru-agent/logs"
)

var httpClient *http.Client

func init() {
	httpClient = &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives:   true,
			MaxIdleConnsPerHost: 1,
		},
	}
}

func UrlJoin(strs ...string) string {
	ss := make([]string, len(strs))
	for i, s := range strs {
		if i == 0 {
			ss[i] = strings.TrimRight(s, "/")
		} else {
			ss[i] = strings.TrimLeft(s, "/")
		}
	}
	return strings.Join(ss, "/")
}

func WritePid(path string) {
	if err := ioutil.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0755); err != nil {
		logs.Assert(err, "Save pid file failed")
	}
}

func MakeDir(p string) error {
	if err := os.MkdirAll(p, 0755); err != nil {
		return err
	}
	return nil
}

func CopyDir(source string, dest string) (err error) {
	// get properties of source dir
	sourceinfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	// create dest dir
	err = os.MkdirAll(dest, sourceinfo.Mode())
	if err != nil {
		return err
	}

	directory, _ := os.Open(source)
	objects, err := directory.Readdir(-1)

	for _, obj := range objects {
		sourcefilepointer := source + "/" + obj.Name()
		destinationfilepointer := dest + "/" + obj.Name()

		if obj.IsDir() {
			// create sub-directories - recursively
			err = CopyDir(sourcefilepointer, destinationfilepointer)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			// perform copy
			err = CopyFile(sourcefilepointer, destinationfilepointer)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
	return
}

func CopyFile(source string, dest string) (err error) {
	sourcefile, err := os.Open(source)
	if err != nil {
		return err
	}

	defer sourcefile.Close()

	destfile, err := os.Create(dest)
	if err != nil {
		return err
	}

	defer destfile.Close()

	_, err = io.Copy(destfile, sourcefile)
	if err == nil {
		sourceinfo, err := os.Stat(source)
		if err != nil {
			err = os.Chmod(dest, sourceinfo.Mode())
		}
	}
	return
}

func Marshal(obj interface{}) []byte {
	bytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		logs.Info("Utils Marshal:", err)
	}
	return bytes
}

func Unmarshal(input io.ReadCloser, obj interface{}) error {
	body, err := ioutil.ReadAll(input)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, obj)
	if err != nil {
		return err
	}
	return nil
}

func GetAppInfo(containerName string) (name string, entrypoint string, ident string) {
	containerName = strings.TrimLeft(containerName, "/")
	appinfo := strings.Split(containerName, "_")
	if len(appinfo) < common.CNAME_NUM {
		return "", "", ""
	}
	l := len(appinfo)
	return strings.Join(appinfo[:l-2], "_"), appinfo[l-2], appinfo[l-1]
}

func Atoi(s string, def int) int {
	if r, err := strconv.Atoi(s); err != nil {
		return def
	} else {
		return r
	}
}

func DoPut(url string) {
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		logs.Debug("Gen request failed", err)
		return
	}
	response, err := httpClient.Do(req)
	if err != nil {
		logs.Debug("Do request failed", err)
		return
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logs.Debug("Read response failed", err)
		return
	}
	logs.Debug("Response:", string(data))
}
