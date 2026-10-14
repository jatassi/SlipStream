package tracker

import "example.com/app/fakes/fake"

var Fake = fake.Spec{Name: "tracker", Online: fake.NoOnlineRun("private trackers get fixtures only")}
