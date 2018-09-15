// +build windows

package win32

import (
	"bytes"
	"fmt"
	"syscall"
	"time"
)

const DefaultMessageTimeout = 1 * time.Minute

type JobObject struct {
	hJob        syscall.Handle
	hCompletion syscall.Handle
}

type JobObjectNotification struct {
	Code               JobObjectMsgCode
	ProcessID          int
	LimitViolationInfo *LimitViolationInfo
}

type JobObjectMsgCode uint32

const (
	JobObjectMsgEndOfJobTime        JobObjectMsgCode = _JOB_OBJECT_MSG_END_OF_PROCESS_TIME
	JobObjectMsgEndOfProcessTime    JobObjectMsgCode = _JOB_OBJECT_MSG_END_OF_PROCESS_TIME
	JobObjectMsgActiveProcessLimit  JobObjectMsgCode = _JOB_OBJECT_MSG_ACTIVE_PROCESS_LIMIT
	JobObjectMsgActiveProcessZero   JobObjectMsgCode = _JOB_OBJECT_MSG_ACTIVE_PROCESS_ZERO
	JobObjectMsgNewProcess          JobObjectMsgCode = _JOB_OBJECT_MSG_NEW_PROCESS
	JobObjectMsgExitProcess         JobObjectMsgCode = _JOB_OBJECT_MSG_EXIT_PROCESS
	JobObjectMsgAbnormalExitProcess JobObjectMsgCode = _JOB_OBJECT_MSG_ABNORMAL_EXIT_PROCESS
	JobObjectMsgProcessMemoryLimit  JobObjectMsgCode = _JOB_OBJECT_MSG_PROCESS_MEMORY_LIMIT
	JobObjectMsgJobMemoryLimit      JobObjectMsgCode = _JOB_OBJECT_MSG_JOB_MEMORY_LIMIT
	JobObjectMsgNotificationLimit   JobObjectMsgCode = _JOB_OBJECT_MSG_NOTIFICATION_LIMIT
)

type LimitViolationInfo struct {
	JobTimeViolation     *LimitViolation
	CPURateViolation     *LimitViolation
	MemoryViolation      *LimitViolation
	HighMemoryViolation  *LimitViolation
	LowMemoryViolation   *LimitViolation
	IOReadBytesViolation *LimitViolation
	IOWriteByesViolation *LimitViolation
	IORateViolation      *LimitViolation
	NetRateViolation     *LimitViolation
}

func (v *LimitViolationInfo) String() string {
	buf := &bytes.Buffer{}
	if v.JobTimeViolation != nil {
		buf.WriteString(fmt.Sprintf("Job Time Violation %v\n", v.JobTimeViolation))
	}
	if v.CPURateViolation != nil {
		buf.WriteString(fmt.Sprintf("CPU Rate Violation %v\n", v.CPURateViolation))
	}
	if v.MemoryViolation != nil {
		buf.WriteString(fmt.Sprintf("Memory Violation %v\n", v.MemoryViolation))
	}
	if v.HighMemoryViolation != nil {
		buf.WriteString(fmt.Sprintf("High Memory Violation %v\n", v.HighMemoryViolation))
	}
	if v.LowMemoryViolation != nil {
		buf.WriteString(fmt.Sprintf("Low Memory Violation %v\n", v.LowMemoryViolation))
	}
	if v.IOReadBytesViolation != nil {
		buf.WriteString(fmt.Sprintf("IO Read Bytes Violation %v\n", v.IOReadBytesViolation))
	}
	if v.IOWriteByesViolation != nil {
		buf.WriteString(fmt.Sprintf("IO Write Bytes Violation %v\n", v.IOWriteByesViolation))
	}
	if v.IORateViolation != nil {
		buf.WriteString(fmt.Sprintf("IO Rate Violation %v\n", v.IORateViolation))
	}
	if v.NetRateViolation != nil {
		buf.WriteString(fmt.Sprintf("Net Rate Violation %v\n", v.NetRateViolation))
	}
	return fmt.Sprintf("violation info: \n%s", buf.String())
}

type LimitViolation struct {
	Measured uint64
	Limit    uint64
}

func (v *LimitViolation) String() string {
	return fmt.Sprintf("measured=%d limit=%d", v.Measured, v.Limit)
}

type JobObjectInformationSetter interface {
	SetJobInfo(hJob syscall.Handle) error
}
type JobObjectInformationGetter interface {
	GetJobInfo(hJob syscall.Handle) error
}

func (j *JobObject) Assign(p *Process) error {
	if p.Pid() == 0 {
		return fmt.Errorf("JobObject.Assign: process has no PID. Is it running?")
	}
	hProc, err := syscall.OpenProcess(_PROCESS_ALL_ACCESS, true, p.Pid())
	if err != nil {
		return err
	}
	defer CloseHandleLogErr(hProc, "win32: failed to close process handle")
	return assignProcessToJobObject(j.hJob, hProc)
}

func (j *JobObject) Close() error {
	if j.hCompletion != 0 {
		defer syscall.Close(j.hCompletion)
	}
	return syscall.Close(j.hJob)
}

func (j *JobObject) SetInformation(info JobObjectInformationSetter) error {
	return info.SetJobInfo(j.hJob)
}

func (j *JobObject) GetInformation(info JobObjectInformationGetter) error {
	return info.GetJobInfo(j.hJob)
}

func (j *JobObject) PollNotifications() (*JobObjectNotification, error) {
	if j.hCompletion != 0 {
		return getQueuedCompletionStatus(j.hJob, j.hCompletion)
	}
	return nil, nil
}

func CreateJobObject(name string) (*JobObject, error) {
	hJob, err := createJobObject(nil, name)
	if err != nil {
		return nil, err
	}
	hCompletionPort, err := syscall.CreateIoCompletionPort(syscall.InvalidHandle, 0, 0, 1)
	if err != nil {
		syscall.Close(hJob)
		return nil, err
	}
	if err = assignJobIOCompletionPort(hJob, hCompletionPort); err != nil {
		syscall.Close(hJob)
		syscall.Close(hCompletionPort)
		return nil, err
	}
	return &JobObject{hJob: hJob, hCompletion: hCompletionPort}, nil
}
