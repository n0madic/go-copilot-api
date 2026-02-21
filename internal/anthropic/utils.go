package anthropic

func MapOpenAIStopReasonToAnthropic(finishReason string) any {
	switch finishReason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	case "content_filter":
		return "end_turn"
	case "", "null":
		return nil
	default:
		return nil
	}
}
