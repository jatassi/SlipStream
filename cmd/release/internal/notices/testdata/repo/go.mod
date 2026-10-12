module example.com/app

go 1.27

require (
	example.com/bsd v1.0.0
	example.com/odd v1.0.0
	example.com/winonly v0.3.0
)

replace (
	example.com/bsd => ./third_party/bsd
	example.com/odd => ./third_party/odd
	example.com/winonly => ./third_party/winonly
)
