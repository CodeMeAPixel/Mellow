package discord

import "fmt"

const plusHistoryLen = 30

func plusOnly(feature string) string {
	return fmt.Sprintf("%s is a Mellow Plus feature. See what Plus offers with `/upgrade`. Everything about safety, coping tools, and your data stays free.", feature)
}
