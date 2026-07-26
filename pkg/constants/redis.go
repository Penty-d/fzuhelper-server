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

package constants

const (
	RedisSlowQuery = 10 // ms redis默认的慢查询时间，适用于 logger
)

const (
	// CourseCacheMaxNum CourseList储存最新TopN个学期
	CourseCacheMaxNum = 2
	// KeyNeverExpire key 永不过期
	KeyNeverExpire = 0
)

// Expire Time
const (
	ClassroomKeyExpire          = 2 * ONE_DAY     // [classroom] 空教室
	LaunchScreenKeyExpire       = 2 * ONE_DAY     // [launch_screen] 开屏页
	UserInfoKeyExpire           = 1 * ONE_WEEK    // [user] 用户信息
	CommonTermListKeyExpire     = 1 * ONE_WEEK    // [common] 学期列表
	CourseTermsKeyExpire        = 3 * ONE_DAY     // [course] 学期列表
	TermInfoKeyExpire           = 7 * ONE_DAY     // [common] 学期详细信息
	ExamRoomKeyExpire           = 10 * ONE_MINUTE // [classroom] 考场信息
	PaperFileDirKeyExpire       = 2 * ONE_DAY     // [paper] 历年卷文件目录
	AcademicScoresExpire        = 5 * ONE_MINUTE  // [academic] 成绩信息
	VisitExpire                 = 1 * ONE_DAY     // [version]访问统计
	LocateDateExpire            = 1 * ONE_HOUR    // [course] 定位日期
	UserInvitationCodeKeyExpire = 1 * ONE_DAY     // [user] 邀请码
	UserFriendKeyExpire         = 3 * ONE_DAY     // [user] 好友列表
	AutoAdjustCourseKeyExpire   = 1 * ONE_DAY     // [common] 调课信息
)

// Key Name
const (
	TermListKey                   = "term_list"                    // [common]
	ContributorJwchKey            = "contributor:jwch"             // [common]
	ContributorYJSYKey            = "contributor:yjsy"             // [common]
	ContributorFzuhelperAppKey    = "contributor:fzuhelper-app"    // [common]
	ContributorFzuhelperServerKey = "contributor:fzuhelper-server" // [common]
	LastLaunchScreenIdKey         = "last_launch_screen_id"        // [launch_screen]
	LocateDateKey                 = "locateDate"                   // [course]
)

// Key Format 动态拼接的 redis key 格式，配合 fmt.Sprintf 使用；集中定义避免各服务散落硬编码
const (
	TermsKeyFormat          = "terms:%s"        // [course] 学期列表，参数: stuId
	CourseListKeyFormat     = "course:%s:%s"    // [course] 课程列表，参数: stuId, term
	UserFriendsKeyFormat    = "user_friends:%s" // [user] 好友列表，参数: stuId
	InvitationCodeKeyFormat = "codes:%s"        // [user] 邀请码，参数: stuId
	CodeMappingKeyFormat    = "code_mapping:%s" // [user] 邀请码映射，参数: code
	ScoresKeyFormat         = "scores:%s"       // [academic] 成绩，参数: stuId
)

// DB Name
const (
	RedisDBEmptyRoom    = 0
	RedisDBLaunchScreen = 1
	RedisDBPaper        = 2
	RedisDBUser         = 3
	RedisDBCommon       = 4
	RedisDBCourse       = 5
	RedisDBAcademic     = 6
	RedisDBVersion      = 7
	RedisDBOA           = 8
)
