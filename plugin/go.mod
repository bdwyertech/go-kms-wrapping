module github.com/hashicorp/go-kms-wrapping/plugin/v2

go 1.16

require (
	github.com/hashicorp/go-hclog v0.16.2
	github.com/hashicorp/go-kms-wrapping/v2 v2.0.0-20211004181108-59533a548d29
	github.com/hashicorp/go-kms-wrapping/wrappers/aead/v2 v2.0.0-20211004181156-d323e1064fea
	github.com/hashicorp/go-plugin v1.4.2
	github.com/stretchr/testify v1.7.0
	google.golang.org/grpc v1.38.0
	google.golang.org/protobuf v1.26.0
)
