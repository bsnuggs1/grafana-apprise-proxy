# grafana-apprise-proxy

Simple go proxy server that converts the data from a grafana webhook into one that can be consumed by apprise.  I just needed something small, simple, and quick that could do this for me since grafana's webhooks does not support custom JSON.

Just sit this guy in front your [apprise](https://github.com/caronc/apprise) instance and you'll be good to go.

### How it Works 

The proxy works by looking for the DashboardId in the json (would be better to implement a schema to validate the structure...but works fine for my purposes) from grafana.  Example alert JSON from grafana:
```
{
    "dashboardId": 1,
    "evalMatches": [
        {
            "value": 1,
            "metric": "Count",
            "tags": {}
        }
    ],
    "imageUrl": "https://grafana.com/static/assets/img/blog/mixed_styles.png",
    "message": "Notification Message",
    "orgId": 1,
    "panelId": 2,
    "ruleId": 1,
    "ruleName": "Panel Title alert",
    "ruleUrl": "http://localhost:3000/d/hZ7BuVbWz/test-dashboard?fullscreen\u0026edit\u0026tab=alert\u0026panelId=2\u0026orgId=1",
    "state": "alerting",
    "tags": {
        "tag name": "tag value"
    },
    "title": "[Alerting] Panel Title alert"
}
```
If the DashboardId is present, the proxy assumes we need to convert it into the json format for apprise, which looks like this if we assume the data based upon the example above:
```
{
    "title": "[Alerting] Panel Title alert"
    "body": "Notification Message"
    "type": "failure"
}
```

If you need to change the mapping, you can modify the ```updatePayload``` method.

## Getting Started

Once you've cloned the project, you will need to create your configuration file, ```conf.yml```.  Inside the file you need to include the following:
```
port: 1450 # Optional, but will default to 1450
url: "http://<apprise hostname>:<port>" # Necessary
```

Alternatively, you can use environment variables.  You just need to set the two vars:
```
GRAFANA_APPRISE_PROXY_TARGET_PORT
GRAFANA_APPRISE_PROXY_TARGET_URL
```

Next you'll need to build the executable, ```grafana-apprise-proxy.exe```.
```
go get -d -v
go build
```

Always make sure the ```conf.yml``` file is with the executable file. Alternatively, the proxy can grab the configuration file from ```/etc/grafana-apprise-proxy/conf.yml```.

### Dockerfile

If you wish to make a container of this application, simply run the following command:

```
docker build --pull --rm -f "Dockerfile" -t grafana-apprise-proxy:latest "."
```

This results in a **12.4MB** image!



