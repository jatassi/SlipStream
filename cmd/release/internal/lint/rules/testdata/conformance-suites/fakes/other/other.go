package other

import "example.com/app/fakes/fake"

const suite = "other"

var Fake = fake.Spec{Name: "other", Online: fake.OnlineSuite(suite)}
