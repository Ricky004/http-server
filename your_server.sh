set -e
tmpFile=$(mktemp)
go build -o "$tmpFile" *.go
exec "$tmpFile" "$@"