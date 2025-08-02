#!/bin/bash
GOOS=linux GOARCH=arm GOARM=7 go build -o bin/golte .

# cleanup ssh
ssh admin@192.168.1.80 << EOF
rm golte
EOF

scp bin/golte admin@192.168.1.80:~/

rm bin/golte

ssh admin@192.168.1.80 << EOF
./golte
EOF
