model:
    goctl model mysql datasource --dir ./internal/model 
        --table article 
        --cache true 
        --url "root:root@tcp(127.0.0.1:3306)/beyond_article" 

rpc:
    goctl rpc protoc ./like.proto --go_out=. --go-grpc_out=. --zrpc_out=./  

api:
    goctl api go 
        --dir=./ 
        --api article.api 