module 3d-api

go 1.26.5

tool honnef.co/go/tools/cmd/staticcheck

// direct depenencies
require (
	github.com/aws/aws-sdk-go-v2 v1.43.5
	github.com/aws/aws-sdk-go-v2/config v1.32.36
	github.com/aws/aws-sdk-go-v2/credentials v1.19.35
	github.com/aws/aws-sdk-go-v2/service/s3 v1.107.1
)

// indirect dependencies
require (
	github.com/BurntSushi/toml v1.4.1-0.20240526193622-a339e1f7089c // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.17 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.36 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.36 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.36 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.37 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.29 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.36 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.37 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.5.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.33.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.38.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.45.5 // indirect
	github.com/aws/smithy-go v1.27.7 // indirect
	golang.org/x/exp/typeparams v0.0.0-20231108232855-2478ac86f678 // indirect
	golang.org/x/mod v0.31.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/tools v0.40.1-0.20260108161641-ca281cf95054 // indirect
	honnef.co/go/tools v0.7.0 // indirect
)
