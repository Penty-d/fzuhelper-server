/*
Copyright 2024 The west2-online Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDurationUntilNextDaily(t *testing.T) {
	locations := []*time.Location{
		time.UTC,
		time.Local,
		time.FixedZone("CST", 8*60*60),
	}
	cases := []struct {
		hour   int
		minute int
	}{
		{0, 10},
		{4, 0},
		{23, 59},
	}
	for _, loc := range locations {
		for _, c := range cases {
			d := DurationUntilNextDaily(c.hour, c.minute, loc)
			// 结果必须为正且不超过 24 小时
			assert.Greater(t, d, time.Duration(0))
			assert.LessOrEqual(t, d, 24*time.Hour)
			// 现在时间加上返回时长应正好落在 loc 时区的 hour:minute
			next := time.Now().In(loc).Add(d)
			assert.Equal(t, c.hour, next.Hour())
			assert.Equal(t, c.minute, next.Minute())
		}
	}
}
