package utils

import (
	"os"
	"testing"
)

func Test_GetAppInfo(t *testing.T) {
	var containerName string
	containerName = "test_1234_abc"
	appname, entrypoint, ident := GetAppInfo(containerName)
	if appname != "test" {
		t.Error("Get appname failed")
	}
	if entrypoint != "1234" {
		t.Error("Get entrypoint failed")
	}
	if ident != "abc" {
		t.Error("Get ident failed")
	}
	containerName = "eru_test_flask_1234_abc"
	appname, entrypoint, ident = GetAppInfo(containerName)
	if appname != "eru_test_flask" {
		t.Error("Get appname failed")
	}
	if entrypoint != "1234" {
		t.Error("Get entrypoint failed")
	}
	if ident != "abc" {
		t.Error("Get ident failed")
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
