# scrapper-microservice

### start
```bash
go run main.go
```
### build
```bash
go build main.go
```
### config
config file path
```bash
/opt/gaetano.yaml
```
```yaml
---
config:
  mongoURI: mongodb://localhost:27017
  database: testing
  maxWorkers: 5
  sleepDuration: 60
```
## Deployment
After compiling the code Deploy the binary on the server in `/usr/local/bin/<name>`
#### SystemD
Create the service file entry in `/etc/systemd/system/<name>.service`
```bash
[Unit]
Description=Gaetano Scraper(GoApp)
After=multi-user.target

[Service]
User=root
Group=root
ExecStart=/usr/local/bin/<name>
StandardOutput=file:/opt/stdlogs.txt
StandardError=file:/opt/errorlogs.txt

[Install]
WantedBy=multi-user.target
```
restart systemd daemon
```bash
sudo systemctl daemon-reload
```
start systemd service
```bash
sudo systemctl start <name-of-service>
```
restart systemd service
```bash
sudo systemctl restart <name-of-service>
```
stop systemd service
```bash
sudo systemctl stop <name-of-service>
```
## Logs
After starting, App will create 2 log files entries for StandardOutput & StandardError in path `/opt/stdlogs.txt` and `/opt/errorlogs.txt` respectively. 
## Packages
For connecting to mongoDB database. App is using official mongoDB Go drivers. [Docs](https://www.mongodb.com/docs/drivers/go/current/)\
Rest of the packages is from Go standard library. [Docs](https://pkg.go.dev/std)

