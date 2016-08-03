
# Example app to show docker prometheus collector


## How to run

First run the http server so it can listen for events and stream stats from daemon
```
go run main.go
```

Next setup a service so you can see the stats

```
docker swarm init
docker service create --name nginx --replicas 3 nginx
```

Wait for the events to get logged as created then 

`curl localhost:8080/metrics`

You should see the metrics. You can scale up/down and see it change. 
