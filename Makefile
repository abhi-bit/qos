all:
	go test -v -coverprofile .testCoverage.txt; go tool cover -html=.testCoverage.txt -o cover.html
