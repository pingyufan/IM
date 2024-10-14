package utils

import (
	"time"
)

type TimeFunc func(interface{}) bool

// delay 首次延迟
// tick 间隔
// fun 定时执行的方法
// param 方法的参数
func Timer(delay, tick time.Duration, fun TimeFunc, param interface{}) {
	go func() {
		if fun == nil {
			return
		}
		t := time.NewTimer(delay)
		for {
			select {
			case <-t.C:
				if fun(param) == false {
					return
				}
				t.Reset(tick)
			}
		}
	}()
}
