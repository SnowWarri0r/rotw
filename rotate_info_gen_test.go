package rotw

import (
	"fmt"
	"testing"
	"time"
)

func Test_AddRotateRule(t *testing.T) {
	err := AddRotateRule("2day", time.Hour*24*2, func() string {
		return "." + time.Now().Format("2006-01") + "-" + fmt.Sprintf("%02d", nowFunc().Day()/2*2)
	})
	if err != nil {
		t.Error(err)
	}
}
