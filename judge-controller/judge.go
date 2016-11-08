package controller

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/VOID001/D-judge/config"
	"github.com/docker/engine-api/client"
	"github.com/pkg/errors"
)

const (
	ExitAC = 42
	ExitWA = 43
)

func (w *Worker) judge(ctx context.Context, rank int64, tid int64) (err error) {
	// Create testcase dir, use to store result
	execdir := filepath.Join(w.WorkDir, "execdir")
	testcasedir := filepath.Join(w.WorkDir, fmt.Sprintf("testcase%03d", rank))
	err = os.Mkdir(testcasedir, DirPerm)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Judge error on Run#%d case %d", w.JudgeInfo.SubmitID, rank))
		return
	}
	// Build the judge script
	cli, er := client.NewClient(config.GlobalConfig.DockerServer, "", nil, nil)
	if er != nil {
		err = errors.Wrap(er, fmt.Sprintf("Judge error on Run#%d case %d", w.JudgeInfo.SubmitID, rank))
		return
	}
	cmd := fmt.Sprintf("/bin/bash -c unzip -o compare/%s -d compare", w.JudgeInfo.CompareZip)
	info, er := w.execcmd(ctx, cli, "root", cmd)
	if er != nil {
		err = errors.Wrap(er, fmt.Sprintf("Judge error on Run#%d case %d", w.JudgeInfo.SubmitID, rank))
		return
	}
	if info.ExitCode != 0 {
		err = errors.New(fmt.Sprintf("Judge error on Run#%d case %d, Command %s exit code is non-zero value %d", w.JudgeInfo.SubmitID, rank, cmd, info.ExitCode))
	}

	cmd = fmt.Sprintf("/bin/bash -c compare/build 2> compare/build.err", w.JudgeInfo.CompareZip)
	info, err = w.execcmd(ctx, cli, "root", cmd)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Judge error on Run#%d case %d", w.JudgeInfo.SubmitID, rank))
		return
	}
	if info.ExitCode != 0 {
		err = errors.New(fmt.Sprintf("Judge error on Run#%d case %d, Command %s exit code is non-zero value %d", w.JudgeInfo.SubmitID, rank, cmd, info.ExitCode))
	}

	cmd = fmt.Sprintf("/bin/bash compare/run execdir/testcase.in execdir/testcase.out testcase001 < execdir/program.out 2> compare.err >compare.out")
	info, err = w.execcmd(ctx, cli, "root", cmd)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Judge error on Run#%d case %d", w.JudgeInfo.SubmitID, rank))
		return
	}

	res := config.RunResult{}
	res.RunTime = 0 // This is uneccessary
	res.TestcaseID = tid
	res.JudgingID = w.JudgeInfo.JudgingID

	// Parse system meta and send to output system
	// For Domjudge compability
	// Save for Judge use
	data, er := ioutil.ReadFile(filepath.Join(execdir, "program.meta"))
	if er != nil {
		err = errors.Wrap(er, "judge error")
		return
	}
	res.OutputSystem = fmt.Sprintf("%s", data)

	switch info.ExitCode {
	case ExitWA:
		res.RunResult = config.ResWA
		// Report Accepted
	case ExitAC:
		res.RunResult = config.ResAC
		// Report Wrong Answer
	default:
		err = errors.New(fmt.Sprintf("Judge return unexpected exit code %d", info.ExitCode))
		return
	}

	// Remove execdir for next time use
	oldexecdir := fmt.Sprintf("%s%03d", execdir, rank)
	err = os.Rename(execdir, oldexecdir)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Judge error on Run#%d case %d", w.JudgeInfo.SubmitID, rank))
		return
	}
	return
}
