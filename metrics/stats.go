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

func getLongID(shortID string) (parentName string, longID string, pid int, err error) {
	pat := filepath.Join(devDir, "*", fmt.Sprintf("*%s*", shortID), "tasks")
	a, err := filepath.Glob(pat)
	if err != nil {
		return
	}
	if len(a) != 1 {
		return "", "", 0, fmt.Errorf("Get Long ID Failed %s", shortID)
	}
	contents, err := ioutil.ReadFile(a[0])
	if err != nil {
		return
	}
	dir := filepath.Dir(a[0])
	longID = filepath.Base(dir)
	parentName = filepath.Base(filepath.Dir(dir))
	pid, _ = strconv.Atoi(strings.Split(string(contents), "\n")[0])
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
	defer outr.Close()

	success := make(chan struct{})
	failure := make(chan error)
	go func() {
		// TODO: 防止被err流block, 删掉先, 之后记得补上
		err = common.Docker.StartExec(
			exec.ID,
			docker.StartExecOptions{
				OutputStream: outw,
				Success:      success,
			},
		)
		outw.Close()
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
		for s.Scan() {
			var name string
			var n [8]uint64
			text := s.Text()
			if strings.Index(text, ":") < 1 {
				continue
			}
			ts := strings.Split(text, ":")
			fmt.Sscanf(ts[0], "%s", &name)
			if !strings.HasPrefix(name, common.VLAN_PREFIX) {
				continue
			}
			fmt.Sscanf(ts[1],
				"%d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d",
				&n[0], &n[1], &n[2], &n[3], &d, &d, &d, &d,
				&n[4], &n[5], &n[6], &n[7], &d, &d, &d, &d,
			)
			result[name+".inbytes"] = n[0]
			result[name+".inpackets"] = n[1]
			result[name+".inerrs"] = n[2]
			result[name+".indrop"] = n[3]
			result[name+".outbytes"] = n[4]
			result[name+".outpackets"] = n[5]
			result[name+".outerrs"] = n[6]
			result[name+".outdrop"] = n[7]
		}
		logs.Debug("Container net status", cid, result)
		return
	}
	err = <-failure
	return nil, err
}
