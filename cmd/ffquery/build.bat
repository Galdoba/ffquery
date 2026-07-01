set GOOS=linux
set GOARCH=amd64
go build -o ffquery_linux .
copy /y ffquery_linux \\192.168.31.55\IN\@SCRIPT