// +build windows

package win32

import (
	"bytes"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

func TestJobObject(t *testing.T) {
	job, err := CreateJobObject("testjob")
	if err != nil {
		t.Error("CreateJobObject", err)
	}
	if err = job.SetInformation(&ExtendedLimitInformation{
		KillOnJobClose: true,
	}); err != nil {
		t.Error("ExtendedLimitInformation", err)
	}
	if err = job.SetInformation(&CPURateControlInformation{
		Rate: &CPUMaxRateInformation{
			HardCap: true,
			Rate:    MHzToCPURate(2048),
		},
	}); err != nil {
		t.Error("CPURateControlInformation/MaxRate", err)
	}
	if err = job.SetInformation(&NotificationLimitInformation{
		CPURateLimit: &NotificationRateLimitTolerance{
			Level:    ToleranceLow,
			Interval: ToleranceIntervalShort,
		},
		NetworkRateLimit: &NotificationRateLimitTolerance{
			Level:    ToleranceLow,
			Interval: ToleranceIntervalShort,
		},
		IORateLimit: &NotificationRateLimitTolerance{
			Level:    ToleranceLow,
			Interval: ToleranceIntervalShort,
		},
	}); err != nil {
		t.Error("NotificationLimitInformation", err)
	}
	// if err = job.SetInformation(&IORateControlInformation{
	// 	MaxBandwidth: 10,
	// 	MaxIOPS:      1,
	// }); err != nil {
	// 	t.Error("IORateControlInformation", err)
	// }
	// if err = job.SetInformation(&NetRateControlInformation{
	// 	MaxBandwidth: 1,
	// 	DSCPTag:      1,
	// }); err != nil {
	// 	t.Error("NetRateControlInformation", err)
	// }

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := exec.Command(SetupTestExe(t), "cpu", "30s")
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	token, err := CurrentProcessToken()
	if err != nil {
		t.Fatal("CurrentProcessToken", err)
	}
	defer token.Close()
	rToken, err := token.CreateRestrictedToken(TokenRestrictions{
		DisableMaxPrivilege: true,
		LUAToken:            true,
		DisableSIDs: []string{
			"BUILTIN\\Administrator",
		},
	})
	if err != nil {
		t.Fatal("CreateRestrictedToken", err)
	}

	envs, err := token.Environment(true)
	if err != nil {
		t.Fatal("token.Environment error", err)
	}
	cmd.Env = envs

	proc, err := StartProcess(cmd, AccessToken(rToken), Suspended)
	if err != nil {
		t.Fatal("StartProcess()", err)
	}
	startTime := time.Now()

	pa, sa, err := proc.AffinityMask()
	if err != nil {
		LogTestError(t, proc.Kill())
		t.Fatal(err)
	}
	t.Logf("ProcessAffinity [%b]", pa)
	t.Logf("SystemAffinity  [%b]", sa)
	if err := token.RunAs(func() {
		if err := job.Assign(proc); err != nil {
			LogTestError(t, proc.Kill())
			t.Fatal("job assign failed", err)
			return
		}
		if err := proc.Resume(); err != nil {
			LogTestError(t, proc.Kill())
			t.Fatal("resume thread failed", err)
		}
	}); err != nil {
		t.Fatal("RunAs", err)
	}
	exitCh := make(chan struct{})

	// Read basic accounting information
	go func() {
		for {
			select {
			case <-exitCh:
				return
			case <-time.After(1 * time.Second):
				ba := &JobObjectBasicAndIOAccounting{}
				if err := job.GetInformation(ba); err != nil {
					t.Errorf("JobObjectBasicAndIOAccounting error: %v", err)
				} else {
					dur := time.Since(startTime) * time.Duration(runtime.NumCPU())
					t.Logf(`{"tt":"%v","ut":"%v","kt":"%v","up":%.3f,"kp":%.3f,"rop":%d,"wop":%d,"oop":%d,"rtx":%d,"wtx":%d,"otx":%d}`,
						dur,
						ba.Basic.TotalUserTime,
						ba.Basic.TotalKernelTime,
						float64(ba.Basic.TotalUserTime)/float64(dur)*100,
						float64(ba.Basic.TotalKernelTime)/float64(dur)*100,
						ba.IO.ReadOperationCount,
						ba.IO.WriteOperationCount,
						ba.IO.OtherOperationCount,
						ba.IO.ReadTransferCount,
						ba.IO.WriteTransferCount,
						ba.IO.OtherTransferCount,
					)
				}
			}
		}
	}()

	// Respond to job object notification events
	go func() {
		for {
			select {
			case <-exitCh:
				return
			case <-time.After(1 * time.Second):
				msg, err := job.PollNotifications()
				if err != nil {
					t.Errorf("PollNotifications error: %v", err)
				} else {
					t.Logf("PollNotifications: %d / %v", msg.Code, msg.LimitViolationInfo)
				}
			}
		}
	}()
	result, err := proc.Wait(exitCh)
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Log("rc", result.ExitStatus)
	}
	t.Log("stdout---\n", stdout.String(), "\n---")
	t.Log("stderr---\n", stderr.String(), "\n---")
}
