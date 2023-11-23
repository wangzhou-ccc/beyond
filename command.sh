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

docker:
    创建容器zookeeper:
        docker run -d --name zookeeper --network beyond -p 2181:2181 -t zookeeper
    创建容器kafka:
        docker run -d --name kafka --network beyond -p 9092:9092 -e KAFKA_BROKER_ID=0 -e KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181 -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 -e KAFKA_LISTENERS=PLAINTEXT://0.0.0.0:9092 wurstmeister/kafka
