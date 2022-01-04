module rpc-gateway

go 1.14

require (
	github.com/drone/envsubst v1.0.3
	github.com/fsnotify/fsnotify v1.5.1
	github.com/ghodss/yaml v1.0.0
	github.com/gin-contrib/sessions v0.0.4
	github.com/gin-gonic/gin v1.7.4
	github.com/go-redis/redis v6.15.2+incompatible
	github.com/gogo/protobuf v1.3.2
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible
	github.com/lestrrat-go/strftime v1.0.5 // indirect
	github.com/rs/xid v1.3.0
	github.com/spf13/viper v1.9.0
	github.com/valyala/fasthttp v1.31.0
	github.com/vearne/golib v0.1.5
	go.uber.org/zap v1.17.0
	golang.org/x/net v0.0.0-20211029224645-99673261e6eb
	golang.org/x/sys v0.0.0-20211102192858-4dd72447c267 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20210828152312-66f60bf46e71
	google.golang.org/grpc v1.41.0
	google.golang.org/protobuf v1.27.1
	gorm.io/driver/mysql v1.1.3
	gorm.io/gorm v1.21.12
	k8s.io/api v0.19.0
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v0.19.0
)
