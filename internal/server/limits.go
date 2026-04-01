package server

type Tier string

const (
	TierFree Tier = "free"
	TierPro  Tier = "pro"
)

type Limits struct {
	Tier        Tier
	Description string
}

func LimitsFor(tier string) Limits {
	if tier == "pro" {
		return Limits{Tier: TierPro, Description: "Unlimited projects and events"}
	}
	return Limits{Tier: TierFree, Description: "1 project, 10k events/mo"}
}

func (l Limits) IsPro() bool {
	return l.Tier == TierPro
}
