package metrics

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"../common"
	"../logs"
	"github.com/fsouza/go-dockerclient"
)

var devDir string = ""

func getLongID(shortID string) (parentName string, longID string, pid string, err error) {
	pat := filepath.Join(devDir, "*", fmt.Sprintf("*%s*", shortID), "tasks")
	a, err := filepath.Glob(pat)
	if err != nil {
		return
	}
	if len(a) != 1 {
		return "", "", "", fmt.Errorf("Get Long ID Failed %s", shortID)
	}
	contents, err := ioutil.ReadFile(a[0])
	if err != nil {
		return
	}
	dir := filepath.Dir(a[0])
	longID = filepath.Base(dir)
	parentName = filepath.Base(filepath.Dir(dir))
	pid = strings.Split(string(contents), "\n")[0]
	return
}

func GetNetStats(cid string) (result map[string]uint64, err error) {
	var exec *docker.Exec
	exec, err = common.Docker.CreateExec(
		docker.CreateExecOptions{
			AttachStdout: true,
			Cmd: []string{
				"cat", "/proc/net/dev",
			},
			Container: cid,
		},
	)
	if err != nil {
		return
	}
	logs.Debug("Create exec id", exec.ID)
	outr, outw := io.Pipe()
	errr, errw := io.Pipe()
	defer outr.Close()
	defer errr.Close()

	success := make(chan struct{})
	failure := make(chan error)
	go func() {
		err = common.Docker.StartExec(
			exec.ID,
			docker.StartExecOptions{
				OutputStream: outw,
				ErrorStream:  errw,
				Success:      success,
			},
		)
		outw.Close()
		errr.Close()
		if err != nil {
			close(success)
			failure <- err
		}
	}()
	if _, ok := <-success; ok {
		success <- struct{}{}
		result = map[string]uint64{}
		s := bufio.NewScanner(outr)
		var d uint64
		for i := 0; s.Scan(); {
			var name string
			var n [8]uint64
			text := s.Text()
			if strings.Index(text, ":") < 1 {
				continue
			}
			ts := strings.Split(text, ":")
			fmt.Sscanf(ts[0], "%s", &name)
			if strings.HasPrefix(name, "veth") || name == "lo" {
				continue
			}
			fmt.Sscanf(ts[1],
				"%d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d",
				&n[0], &n[1], &n[2], &n[3], &d, &d, &d, &d,
				&n[4], &n[5], &n[6], &n[7], &d, &d, &d, &d,
			)
			j := "." + strconv.Itoa(i)
			result["inbytes"+j] = n[0]
			result["inpackets"+j] = n[1]
			result["inerrs"+j] = n[2]
			result["indrop"+j] = n[3]
			result["outbytes"+j] = n[4]
			result["outpackets"+j] = n[5]
			result["outerrs"+j] = n[6]
			result["outdrop"+j] = n[7]
			i++
		}
		logs.Debug("Container net status", cid, result)
		return
	}
	err = <-failure
	return
}
