# Album Remixing tool for the Swandive music archive
This website uses the Go programming language with WASM as a compilation target.
The templating is performed with HTML, as seen in /assets/index.html.
No Javascript is used except for the `wasm_exec.js` functions that are necessary to load WASM.

## To build the Go code
```
cd ./src/wasm
make build
```
## To edit the HTML template
`cd ./assets`
Edit the index.html file
## To serve the page on port 9090
```
cd ./src/server
go run main.go
```
## To update the `albums.json` with the contents of S3/R2
```
aws s3api list-objects-v2 --endpoint-url <OBJECT_BASE_URL> --bucket <BUCKET_NAME> | ./s3-to-albums.sh > <OUTPUT_FILE>

//EXAMPLE
aws s3api list-objects-v2 --endpoint-url https://bde77e730d345a8dbb818bf3633633b3.r2.cloudflarestorage.com --bucket swandive-archive | ./s3-to-albums.sh > ./src/wasm/albums.json
```
