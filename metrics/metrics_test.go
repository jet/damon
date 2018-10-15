package metrics

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
	"time"
)

type testSensor struct {
	StartTime  time.Time
	Cores      int
	TimeScale  float64
	KernelPerc float64
	UserPerc   float64
}

func (s testSensor) MeasureCPU() CPUMeasurement {
	d := time.Duration(float64(time.Since(s.StartTime)) * s.TimeScale)
	return CPUMeasurement{
		TotalTime:  d,
		KernelTime: time.Duration(float64(d) * s.KernelPerc * float64(s.Cores)),
		UserTime:   time.Duration(float64(d) * s.UserPerc * float64(s.Cores)),
	}
}

func TestCPUSampler(t *testing.T) {
	err := quick.Check(func(kernPerc float64, userPerc float64, cores int64, mhz float64) bool {
		sensor := testSensor{
			StartTime:  time.Now(),
			Cores:      int(cores),
			TimeScale:  1000.0,
			KernelPerc: kernPerc,
			UserPerc:   userPerc,
		}
		s := &CPUCollector{
			Cores:      int(cores),
			MHzPerCore: mhz,
		}
		// nolint
		for i := 0; i < 5; i++ {
			select {
			case <-time.After(1 * time.Millisecond):
				sample := s.Sample(sensor.MeasureCPU())
				if math.Abs(sample.UserPercent-userPerc) > 0.01 {
					t.Errorf("user percent expected delta too great: expected=%.5f, actual=%.5f", userPerc, sample.UserPercent)
					return false
				}
				if math.Abs(sample.KernelPercent-kernPerc) > 0.01 {
					t.Errorf("kernel percent expected delta too great: expected=%.5f, actual=%.5f", userPerc, sample.UserPercent)
					return false
				}
			}
		}
		return true
	}, &quick.Config{
		Values: func(v []reflect.Value, r *rand.Rand) {
			v[0] = reflect.ValueOf(r.Float64())
			v[1] = reflect.ValueOf(r.Float64())
			v[2] = reflect.ValueOf(1 + r.Int63n(64))
			v[3] = reflect.ValueOf(1000 + r.Float64()*2000)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}
