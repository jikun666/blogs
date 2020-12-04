原线上服务使用的是 tcp check 方式，通过注册时指定：

```golang
curl -H "X-Consul-Token: xxxxxx" -XPUT "http://$ip:8500/v1/agent/service/register" --data '{
    "name": "dump",
    "tags": ["happy"],
    "address": "'$ip'",
    "port": 9085,
    "checks": [{"tcp": "127.0.0.1:9085", "interval": "5s"}],
    "weights": {"passing": '$passing', "warning": 1}
}' && echo "ok"
```

consul 一共提供了 7 种健康检查的方式：

- script check
- http check
- tcp check
- ttl check
- docker check
- grpc check
- alias check

script check:
```json
{
  "check": {
    "id": "mem-util",
    "name": "Memory utilization",
    "args": [
      "/bin/sh",
      "-c",
      "/usr/bin/free | awk '/Mem/{printf($3/$2*100)}' | awk '{ print($0); if($1 > 70) exit 1;}'"
    ],
    "interval": "10s",
    "timeout": "1s"
  }
}
```
根据 args 的执行结果作为检查结果，0 通过，1 警告，其他 未通过。（上面的检查内存占用）

http check:
```json
{
  "check": {
    "id": "dashboard_check",
    "name": "Check Dashboard health 5001",
    "service_id": "dashboard_1",
    "http": "http://localhost:5001/health",
    "method": "GET",
    "interval": "10s",
    "timeout": "1s"
  }
}
```
需要服务提供一个5001端口，根据http状态码作为检查结果，返回 2xx 才算通过。


tcp check:
```json
{
  "check": {
    "id": "counting_check",
    "name": "Check Counter health 5000",
    "service_id": "counting_1",
    "tcp": "localhost:5000",
    "interval": "10s",
    "timeout": "1s"
  }
}
```
consul 会与 5000 端口建立连接，如果连接可以被 accept 算通过，否则不通过。

ttl check:
```json
{
  "check": {
    "id": "web-app",
    "name": "Web App Status",
    "notes": "Web app does a curl internally every 10 seconds",
    "ttl": "30s"
  }
}
```
以上在 30 秒内服务必须向 consul 发起 put 请求，及时报告自己仍然健康，否则会被认为健康检查不通过。在官方 api 库中应调用的方法为：

```golang
// UpdateTTL is used to update the TTL of a check. This uses the newer API
// that was introduced in Consul 0.6.4 and later. We translate the old status
// strings for compatibility (though a newer version of Consul will still be
// required to use this API).
func (a *Agent) UpdateTTL(checkID, output, status string) error {
	switch status {
	case "pass", HealthPassing:
		status = HealthPassing
	case "warn", HealthWarning:
		status = HealthWarning
	case "fail", HealthCritical:
		status = HealthCritical
	default:
		return fmt.Errorf("Invalid status: %s", status)
	}

	endpoint := fmt.Sprintf("/v1/agent/check/update/%s", checkID)
	r := a.c.newRequest("PUT", endpoint)
	r.obj = &checkUpdate{
		Status: status,
		Output: output,
	}

	_, resp, err := requireOK(a.c.doRequest(r))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
```

alias check:
```json
{
  "check": {
    "id": "counter-alias",
    "name": "counter_alias",
    "service_id": "dashboard_1",
    "alias_service": "counting_1"
  }
}
```
前端服务 dashboard_1 依赖后端服务 counting_1，这里想让前端服务的健康检查关注后端服务的健康检查，所以直接用了一个 alias check。


官网地址：
- https://www.consul.io/docs/discovery/checks
- https://learn.hashicorp.com/tutorials/consul/service-registration-health-checks