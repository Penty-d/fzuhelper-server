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

package pack

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/west2-online/jwch"
	"github.com/west2-online/yjsy"
)

func TestNormalizeScore(t *testing.T) {
	Convey("normalizeScore", t, func() {
		Convey("should normalize special score texts", func() {
			So(normalizeScore("成绩尚未录入"), ShouldEqual, "暂无")
			So(normalizeScore("成绩只录一遍"), ShouldEqual, "录入中")
		})

		Convey("should return normal score as is", func() {
			So(normalizeScore("90"), ShouldEqual, "90")
			So(normalizeScore("优秀"), ShouldEqual, "优秀")
		})
	})
}

func TestBuildScores(t *testing.T) {
	Convey("BuildScores", t, func() {
		Convey("should normalize special score texts without modifying source data", func() {
			data := []*jwch.Mark{
				{Name: "数据结构", Score: "90"},
				{Name: "计算机网络", Score: "成绩尚未录入"},
				{Name: "操作系统", Score: "成绩只录一遍"},
			}

			scores := BuildScores(data)

			So(len(scores), ShouldEqual, 3)
			So(scores[0].Score, ShouldEqual, "90")
			So(scores[1].Score, ShouldEqual, "暂无")
			So(scores[2].Score, ShouldEqual, "录入中")
			// 源数据不应被就地改写
			So(data[1].Score, ShouldEqual, "成绩尚未录入")
			So(data[2].Score, ShouldEqual, "成绩只录一遍")
		})
	})
}

func TestBuildScoresYjsy(t *testing.T) {
	Convey("BuildScoresYjsy", t, func() {
		Convey("should normalize special score texts without modifying source data", func() {
			data := []*yjsy.Mark{
				{Name: "高等数学", Score: "95"},
				{Name: "矩阵论", Score: "成绩尚未录入"},
				{Name: "随机过程", Score: "成绩只录一遍"},
			}

			scores := BuildScoresYjsy(data)

			So(len(scores), ShouldEqual, 3)
			So(scores[0].Score, ShouldEqual, "95")
			So(scores[1].Score, ShouldEqual, "暂无")
			So(scores[2].Score, ShouldEqual, "录入中")
			// 源数据不应被就地改写（该切片会被 service 层异步写入缓存）
			So(data[1].Score, ShouldEqual, "成绩尚未录入")
			So(data[2].Score, ShouldEqual, "成绩只录一遍")
		})
	})
}
