package rotw

import (
	"reflect"
	"testing"
	"time"
)

func Test_SetNowFunc(t *testing.T) {
	setNowFunc(func() time.Time {
		return time.Now().Add(time.Hour)
	})
	if nowFunc() == time.Now() {
		t.Error("nowFunc should be changed")
	}
	setNowFunc(time.Now)
	if reflect.ValueOf(nowFunc).Pointer() != reflect.ValueOf(time.Now).Pointer() {
		t.Error("nowFunc should be changed")
	}
}
