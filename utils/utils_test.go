package utils

import (
	"os"
	"testing"

	"../common"
)

func Test_GetAppInfo(t *testing.T) {
	var containerName string
	containerName = "test_1234"
	appname, appid, apptype := GetAppInfo(containerName)
	if appname != "test" {
		t.Error("Get appname failed")
	}
	if appid != "1234" {
		t.Error("Get appid failed")
	}
	if apptype != common.DEFAULT_TYPE {
		t.Error("Get apptype failed")
	}
}

func Test_UrlJoin(t *testing.T) {
	strs := []string{"http://a.b.c", "d", "e"}
	ss := UrlJoin(strs...)
	if ss != "http://a.b.c/d/e" {
		t.Error("Join invaild")
	}
}

func Test_WritePid(t *testing.T) {
	p := "/tmp/test.pid"
	WritePid(p)
	defer os.RemoveAll(p)
	if _, err := os.Stat(p); err != nil {
		t.Error(err)
	}
}

func Test_CopyDir(t *testing.T) {
	defer func() {
		os.RemoveAll("/tmp/t1")
		os.RemoveAll("/tmp/t2")
	}()
	if err := MakeDir("/tmp/t1"); err != nil {
		t.Error(err)
	}
	if err := CopyDir("/tmp/t1", "/tmp/t2"); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat("/tmp/t2"); err != nil {
		t.Error(err)
	}
}
