#!/bin/bash
GOOS=linux GOARCH=arm GOARM=7 go build -o bin/golte .

# Disable echo
set +x

scp bin/golte admin@192.168.1.80:~/
scp config.yaml admin@192.168.1.80:~/

ssh admin@192.168.1.80 << EOF
sudo mv golte /opt/golte/golte
sudo mv config.yaml /opt/golte/config.yaml
sudo systemctl restart golte.service
EOF
