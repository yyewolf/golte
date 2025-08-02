#!/bin/bash
GOOS=linux GOARCH=arm GOARM=7 go build -o golte .

# cleanup ssh
ssh admin@192.168.1.80 << EOF
rm golte
EOF

scp golte admin@192.168.1.80:~/

rm golte

ssh admin@192.168.1.80 << EOF
./golte
EOF
