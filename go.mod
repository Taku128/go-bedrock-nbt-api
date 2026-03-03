module github.com/ntaku256/go-bedrock-nbt-api

go 1.24.0

replace github.com/ntaku256/go-bedrock-nbt-converter => ../go-bedrock-nbt-converter

require (
	github.com/Tnze/go-mc v1.20.2
	github.com/aws/aws-lambda-go v1.52.0
	github.com/ntaku256/go-bedrock-nbt-converter v0.0.0-00010101000000-000000000000
	github.com/uberswe/mcnbt v0.1.3
)

require (
	github.com/df-mc/goleveldb v1.1.9 // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/klauspost/compress v1.18.1 // indirect
	github.com/sandertv/gophertunnel v1.54.0 // indirect
)

replace github.com/uberswe/mcnbt => ../mcnbt_code/mcnbt-main
