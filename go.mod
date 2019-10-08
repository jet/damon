module github.com/jet/damon

require (
	github.com/LK4D4/joincontext v0.0.0-20171026170139-1724345da6d5 // indirect
	github.com/Microsoft/go-winio v0.4.13 // indirect
	github.com/StackExchange/wmi v0.0.0-20180116203802-5d049714c4a6 // indirect
	github.com/fsnotify/fsnotify v1.4.7 // indirect
	github.com/go-ole/go-ole v1.2.1 // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/gorhill/cronexpr v0.0.0-20180427100037-88b0669f7d75 // indirect
	github.com/hashicorp/consul/api v1.1.0 // indirect
	github.com/hashicorp/go-hclog v0.9.2
	github.com/hashicorp/go-immutable-radix v1.1.0 // indirect
	github.com/hashicorp/go-version v1.2.0 // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/hashicorp/memberlist v0.1.4 // indirect
	github.com/hashicorp/nomad v0.9.5
	github.com/hashicorp/raft v1.1.1 // indirect
	github.com/hashicorp/serf v0.8.3 // indirect
	github.com/hashicorp/vault/api v1.0.4 // indirect
	github.com/hpcloud/tail v1.0.0 // indirect
	github.com/miekg/dns v1.1.15 // indirect
	github.com/mitchellh/hashstructure v1.0.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.2
	github.com/rs/zerolog v1.9.1
	github.com/shirou/gopsutil v2.18.12+incompatible // indirect
	github.com/shirou/w32 v0.0.0-20160930032740-bb4de0191aa4 // indirect

	// Matching https://github.com/hashicorp/nomad/blob/v0.9.5/vendor/vendor.json#L370
	// from github.com/hashicorp/nomad@v0.9.5
	github.com/ugorji/go v0.0.0-20170620060102-0053ebfd9d0e // indirect
	github.com/zclconf/go-cty v1.1.0 // indirect
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4 // indirect
	golang.org/x/net v0.0.0-20190724013045-ca1201d0de80 // indirect
	golang.org/x/sys v0.0.0-20190730183949-1393eb018365
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/grpc v1.22.1 // indirect
	gopkg.in/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.2.2 // indirect
)

//Fix the forever mess
replace github.com/Sirupsen/logrus v1.4.2 => github.com/sirupsen/logrus v1.4.2

// Nomad uses a custom branch on a custom fork of go-winio
// So in order to make this compile, we have to do the same
// See: https://github.com/hashicorp/nomad/blob/v0.9.4/vendor/vendor.json#L11
replace github.com/Microsoft/go-winio v0.4.13 => github.com/endocrimes/go-winio v0.4.13-0.20190628114223-fb47a8b41948
