package cost

import (
	"math"

	"github.com/hashir500/Fuse/internal/config"
)

type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

func Estimate(usage Usage, costs config.ModelCosts) float64 {
	return (float64(usage.PromptTokens)/1000.0)*costs.InputCostPer1K +
		(float64(usage.CompletionTokens)/1000.0)*costs.OutputCostPer1K
}

func PreflightUsage(usage Usage, estimation config.EstimationConfig) Usage {
	if estimation.Mode != "typical" || usage.CompletionTokens <= 0 {
		return usage
	}

	completionTokens := int(math.Ceil(float64(usage.CompletionTokens) * estimation.OutputRatio))
	if estimation.TypicalOutputTokens > 0 && completionTokens > estimation.TypicalOutputTokens {
		completionTokens = estimation.TypicalOutputTokens
	}
	if completionTokens > usage.CompletionTokens {
		completionTokens = usage.CompletionTokens
	}

	usage.CompletionTokens = completionTokens
	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	return usage
}
