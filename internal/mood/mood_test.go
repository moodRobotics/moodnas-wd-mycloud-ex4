package mood

import (
	"testing"

	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/ai/local"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/platform"
)

func TestEvaluate(t *testing.T) {
	cases := []struct {
		name string
		in   Inputs
		want string
	}{
		{"calm", Inputs{Sensors: platform.Sensors{CPUTempC: 50}, Fill: local.FillPrediction{Learning: true}}, Calm},
		{"sleepy", Inputs{Quiet: true, CPUPct: 3, Fill: local.FillPrediction{Learning: true}}, Sleepy},
		{"busy", Inputs{CPUPct: 85, Fill: local.FillPrediction{Learning: true}}, Focused},
		{"warm", Inputs{Sensors: platform.Sensors{CPUTempC: 78}, Fill: local.FillPrediction{Learning: true}}, Worried},
		{"filling", Inputs{Fill: local.FillPrediction{DaysToFull: 12}}, Worried},
		{"risky disk", Inputs{Risks: []*local.DiskRisk{{Level: "high", Bay: 3}}, Fill: local.FillPrediction{Learning: true}}, Alarmed},
		{"degraded", Inputs{Arrays: []platform.Array{{Name: "md1", State: "degraded"}}, Fill: local.FillPrediction{Learning: true}}, Alarmed},
		{"ransomware beats busy", Inputs{CPUPct: 90, Incidents: []local.Incident{{Share: "x"}}, Fill: local.FillPrediction{Learning: true}}, Alarmed},
	}
	for _, c := range cases {
		if got := Evaluate(c.in); got.State != c.want {
			t.Errorf("%s: got %s want %s (%+v)", c.name, got.State, c.want, got.Reasons)
		}
	}
}
