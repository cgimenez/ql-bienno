mode=""
if [[ $1 != "" ]]; then
  mode="release"
fi
go build -o ql *.go